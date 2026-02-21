package main

import (
	"context"
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
	"github.com/Chintukr2004/pulsestream/internal/store"
	"github.com/Chintukr2004/pulsestream/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
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

	dbURL := "postgres://postgres:1234@localhost:5432/pulsestream"

	dbpool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("unable to connect database: %v", err)
	}

	defer dbpool.Close()

	fmt.Println("Connected to database")

	err = dbpool.Ping(ctx)
	if err != nil {
		log.Fatalf("Database ping failed: %v", err)

	}
	fmt.Println("PulseStream server started..")
	postStore := store.NewPostStore(dbpool)

	postsChan := make(chan model.Post, 100)

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
				postsChan <- post
				time.Sleep(1 * time.Second)
			}

		}
	}()

	var postCount int64

	//worker pool

	workerCount := 3

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					logger.Info("Worker shutting down", "worker_id", id)
					return
				case post := <-postsChan:
					start := time.Now()
					ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
					err := postStore.Insert(ctxTimeout, post)
					cancel()

					duration := time.Since(start)

					if err != nil {
						logger.Error("failed to insert post",
							"worker_id", id,
							"error", err,
						)
						continue
					}
					hub.Broadcast(post)
					atomic.AddInt64(&postCount, 1)
					logger.Info("post inserted",
						"worker_id", id,
						"username", post.Username,
						"duration_ms", duration.Milliseconds())
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
