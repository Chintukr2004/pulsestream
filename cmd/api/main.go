package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Chintukr2004/pulsestream/internal/generator"
	"github.com/Chintukr2004/pulsestream/internal/handler"
	"github.com/Chintukr2004/pulsestream/internal/model"
	"github.com/Chintukr2004/pulsestream/internal/redisclient"
	"github.com/Chintukr2004/pulsestream/internal/store"
	"github.com/Chintukr2004/pulsestream/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup

	dbURL := "postgres://postgres:password@postgres:5432/pulsestream"

	var dbpool *pgxpool.Pool
	var err error

	for i := 0; i < 10; i++ {
		dbpool, err = pgxpool.New(ctx, dbURL)
		if err == nil {
			err = dbpool.Ping(ctx)
			if err == nil {
				break
			}
		}

		logger.Info("waiting for database...", "attempt", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		logger.Error("could not connect to database after retries", "error", err)
		return
	}

	logger.Info("connected to database")

	// fmt.Println("PulseStream server started..")
	postStore := store.NewPostStore(dbpool)

	//redis
	redisClient := redisclient.NewClient()

	const (
		streamName    = "posts_stream"
		consumerGroup = "workers_group"
	)

	hub := ws.NewHub()
	go hub.Run()

	//generator
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				logger.Info("Generator shutting down")
				return
			default:
				post := generator.GeneratePost()
				data, err := json.Marshal(post)
				if err != nil {
					logger.Error("failed to marshal post", "error", err)
					continue
				}

				err = redisclient.AddToStream(ctx, redisClient, streamName, map[string]interface{}{
					"data": string(data),
				})
				if err != nil {
					logger.Error("failed to add to stream", "error", err)
				}
				time.Sleep(1 * time.Second)
			}

		}
	}()

	var postCount int64

	err = redisclient.CreateConsumerGroup(ctx, redisClient, streamName, consumerGroup)
	if err != nil {
		logger.Error("failed to create consumer group", "error", err)
		return
	}

	//worker pool

	workerCount := 3
	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			consumerName := fmt.Sprintf("worker-%d", id)

			for {
				select {
				case <-ctx.Done():
					logger.Info("worker shutting down", "worker_id", id)
					return
				default:
					streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
						Group:    consumerGroup,
						Consumer: consumerName,
						Streams:  []string{streamName, ">"},
						Count:    1,
						Block:    5 * time.Second,
					}).Result()

					if err != nil && err != redis.Nil {
						logger.Error("XReadGroup error", "error", err)
						continue
					}

					for _, stream := range streams {
						for _, message := range stream.Messages {

							payload := message.Values["data"].(string)

							var post model.Post
							err := json.Unmarshal([]byte(payload), &post)
							if err != nil {
								logger.Error("failed to unmarshal", "error", err)
								continue
							}

							start := time.Now()

							ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
							err = postStore.Insert(ctxTimeout, post)
							cancel()

							duration := time.Since(start)

							if err != nil {
								logger.Error("DB insert failed",
									"worker_id", id,
									"error", err,
								)
								continue
							}

							// ACK message
							err = redisClient.XAck(ctx, streamName, consumerGroup, message.ID).Err()
							if err != nil {
								logger.Error("failed to ack message", "error", err)
							}

							hub.Broadcast(post)

							logger.Info("post processed via stream",
								"worker_id", id,
								"duration_ms", duration.Milliseconds(),
							)
						}
					}
				}
			}
		}(i)
	}

	// monitoring goroutine

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count := atomic.SwapInt64(&postCount, 0)
				logger.Info("throuput report",
					"post_last_5s", count,
					"post_per_sec", float64(count)/5,
				)
			}
		}
	}()

	//Handlers

	postHandler := handler.NewPostHandler(postStore)

	http.HandleFunc("/posts", postHandler.GetPosts)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWS(hub, w, r)
	})

	// server setup
	go func() {
		logger.Info("Http server started", "port", 8080)
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}

	}()

	<-sigChan
	logger.Info("Shutdown signal recived...")
	cancel()
	wg.Wait()

	fmt.Println("All goroutines stopped.")
	fmt.Println("Closing DB...")

}
