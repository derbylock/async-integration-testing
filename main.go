package main

import (
	"log"
	"os"
	"strings"

	"github.com/derbylock/async-integration-testing/cmd/server"
	"github.com/derbylock/async-integration-testing/internal/db"
	"github.com/go-redis/redis/v9"
)

const (
	REDIS_ADDRS    = "REDIS_ADDRS"
	REDIS_PASSWORD = "REDIS_PASSWORD"
)

func main() {
	log.Println("Starting HTTP server")

	redisAddrs := requireEnv(REDIS_ADDRS)
	redisPassword := os.Getenv(REDIS_PASSWORD)

	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    strings.Split(redisAddrs, ","),
		Password: redisPassword,
	})
	defer redisClient.Close()

	storage := db.NewRedisStorage(redisClient, db.PROTO_CODEC)
	server := server.NewServer(storage)
	log.Fatal(server.ListenAndServe())
}

func requireEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Printf("the %s environment variable is not specified", name)
		os.Exit(1)
	}
	return value
}
