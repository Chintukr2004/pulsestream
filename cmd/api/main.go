package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Chintukr2004/pulsestream/internal/generator"
	"github.com/Chintukr2004/pulsestream/internal/handler"
	"github.com/Chintukr2004/pulsestream/internal/model"
	"github.com/Chintukr2004/pulsestream/internal/store"
	"github.com/Chintukr2004/pulsestream/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

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
	go func() {
		for {
			post := generator.GeneratePost()
			postsChan <- post
			time.Sleep(1 * time.Second)
		}
	}()

	//worker pool

	workerCount := 3

	for i := 0; i < workerCount; i++ {
		go func(id int) {
			for post := range postsChan {

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				err := postStore.Insert(ctx, post)
				cancel()
				if err != nil {
					continue
				}
				hub.Broadcast(post)
				fmt.Printf("Worker %d inserted post: %s\n", id, post.Content)

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
	select {}

}
