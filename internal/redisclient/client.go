package redisclient

import (
	"context"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

func NewClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Fatal("failed to connect to redis: %v", err)
	}
	log.Println("Connected to redis")

	return client

}

func CreateConsumerGroup(ctx context.Context, client *redis.Client, stream, group string) error {
	err := client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil {
		//BUSYGROUP means already exixts
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return err
		}
	}
	return nil
}

func AddToStream(ctx context.Context, client *redis.Client, stream string, values map[string]interface{}) error {
	return client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Err()
}
