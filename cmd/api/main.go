package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
				fmt.Println("Generator shutting down...")
				return
			default:
				post := generator.GeneratePost()
				postsChan <- post
				time.Sleep(1 * time.Second)
			}

		}
	}()

	//worker pool

	workerCount := 3

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					fmt.Printf("Worker %d shutting down...\n", id)
					return
				case post := <-postsChan:
					ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
					err := postStore.Insert(ctxTimeout, post)
					cancel()
					if err != nil {
						continue
					}
					hub.Broadcast(post)
					// fmt.Printf("Worker %d inserted post: %s\n", id, post.Content)
				}
			}
		}(i)
	}

	postHandler := handler.NewPostHandler(postStore)

	http.HandleFunc("/posts", postHandler.GetPosts)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWS(hub, w, r)
	})

	go func() {
		fmt.Println("Http server running on port:8000")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}

	}()

	<-sigChan
	fmt.Println("Shutdown signal recived...")
	cancel()
	wg.Wait()

	fmt.Println("All goroutines stopped.")
	fmt.Println("Closing DB...")

}
