package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"log"
	"net/http"
	"time"
)

type wallet struct {
	ID           string         `json:"id"`
	Transactions []*transaction `json:"transactions"`
	Balance      float32        `json:"balance"`
}

type transaction struct {
	SourceWallet      string    `json:"source_wallet"`
	DestinationWallet string    `json:"destination_wallet"`
	Amount            float32   `json:"amount"`
	Date              time.Time `json:"date"`
}

func postTransaction(rdb *redis.Client) func(*gin.Context) {
	return func(c *gin.Context) {
		var receivedTransaction transaction
		if errBind := c.BindJSON(&receivedTransaction); errBind != nil {
			log.Printf("could not process received transaction, %s", errBind)
			return
		}
		log.Printf("received Transaction:%+v", receivedTransaction)

		c.IndentedJSON(http.StatusCreated, receivedTransaction)
	}
}
