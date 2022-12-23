package main

import (
	"log"
	"os"

	"github.com/derbylock/async-integration-testing/cmd/server"
)

const (
	REDIS_ADDRS    = "REDIS_ADDRS"
	REDIS_PASSWORD = "REDIS_PASSWORD"
)

func main() {
	log.Println("Starting HTTP server")

	redisAddrs := requireEnv(REDIS_ADDRS)
	redisPassword := os.Getenv(REDIS_PASSWORD)

	server := server.NewRedisBackedServer(redisAddrs, redisPassword)
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
