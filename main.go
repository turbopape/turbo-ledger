package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"

	"github.com/go-redis/redis/v8"
	"log"
	"os"
)

func main() {
	// Recovering after panics, trying to keep the turbo-ledger alive
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Unable to recover from panic in main - reason: %e", err)
		}
	}()

	errEnv := godotenv.Load()
	if errEnv != nil {
		fmt.Println("Could not load .env File, getting values from environment...")
	}

	app := &cli.App{
		Name:    "turbo-ledger - a scalable ledger",
		Usage:   "Provides safe horizontally-scalable ledgers based on Redis",
		Version: "naive-version",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "redisHost",
				Aliases: []string{"rh"},
				Usage:   "Redis Host",
				EnvVars: []string{"TURBO_LEDGER_REDIS_HOST"},
			},
			&cli.StringFlag{
				Name:    "redisUser",
				Aliases: []string{"ru"},
				Usage:   "Redis User",
				EnvVars: []string{"TURBO_LEDGER_REDIS_USER"},
			},
			&cli.StringFlag{
				Name:    "redisPassword",
				Aliases: []string{"rp"},
				Usage:   "Redis Password",
				EnvVars: []string{"TURBO_LEDGER_REDIS_PASSWORD"},
			},

			&cli.StringFlag{
				Name:    "listenAddress",
				Aliases: []string{"la"},
				Usage:   "Listen ddress",
				Value:   ":12000",
				EnvVars: []string{"TURBO_LEDGER_LISTEN_ADDRESS"},
			},
		},
		Action: func(c *cli.Context) error {
			// Connecting to Redis...
			redisHost := c.String("redisHost")
			redisUser := c.String("redisUser")
			redisPassword := c.String("redisPassword")
			listenAddress := c.String("listenAddress")

			log.Printf("Connecting to Redis on Host %s with User %s ... ", redisHost, redisUser)
			rdb := redis.NewClient(&redis.Options{
				Addr:     redisHost,
				Username: redisUser,
				Password: redisPassword, // no password set
				DB:       0,             // use default DB
			})
			if errRedis := rdb.Context().Err(); errRedis != nil {
				return errRedis
			}
			log.Printf("Successfully connected to Redis...")

			// Running the API

			router := gin.Default()
			router.POST("transactions", postTransaction(rdb))
			router.Run(listenAddress)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Panic(err)
	}

}
