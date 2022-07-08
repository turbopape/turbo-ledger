package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
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
				Usage:   "Listen address",
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

			rdb, pong, err := connectToRedis(redisHost, redisUser, redisPassword)
			if err != nil {
				return err
			}
			log.Printf("Successfully connected to Redis %s", pong)

			// Preparing redsync instance
			pool := goredis.NewPool(rdb)
			rs := redsync.New(pool)
			mutexname := "global-wallets-mutex"
			mutex := rs.NewMutex(mutexname)

			// Running the Genesis process, the vault account
			errGenesis := Genesis(rdb, "vault", 500)
			if errGenesis != nil {
				return errGenesis
			}

			// Running the API
			router := gin.Default()

			router.POST("wallets", PostWallet(rdb))
			router.POST("transactions", PostTransaction(rdb, mutex))
			router.GET("wallets", SearchWalletsByTags(rdb))
			router.Run(listenAddress)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Panic(err)
	}

}

func connectToRedis(redisHost string, redisUser string, redisPassword string) (*redis.Client, string, error) {
	ctx := context.Background()
	log.Printf("Connecting to Redis on Host %s with User %s ... ", redisHost, redisUser)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Username: redisUser,
		Password: redisPassword, // no password set
		DB:       0,             // use default DB
	})
	pong, errRedis := rdb.Ping(ctx).Result()
	if errRedis != nil {
		return nil, "", errRedis
	}
	return rdb, pong, nil
}
