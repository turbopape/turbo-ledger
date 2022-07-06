package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"testing"
)

func TestGenesis(t *testing.T) {
	errEnv := godotenv.Load()
	if errEnv != nil {
		fmt.Println("Could not load .env File, getting values from environment...")
	}

	redisHost := os.Getenv("TURBO_LEDGER_REDIS_HOST")
	redisUser := os.Getenv("TURBO_LEDGER_REDIS_USER")
	redisPassword := os.Getenv("TURBO_LEDGER_REDIS_PASSWORD")

	rdb, pong, err := connectToRedis(redisHost, redisUser, redisPassword)
	if err != nil {
		t.Errorf("error connecting to redis, %s %s", pong, err)
	}

	if err := Genesis(rdb, "vault", 4); err != nil {
		t.Errorf("error processing genesis %s", err)
	}
}
