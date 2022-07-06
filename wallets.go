package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"log"
	"net/http"
	"strings"
	"time"
)

var errVaultAccountExist error = errors.New("vault account exists")

type wallet struct {
	ID           string         `json:"wallet_id"`
	Transactions []*transaction `json:"transactions"`
	Balance      float32        `json:"balance"`
}

type transaction struct {
	ID                string    `json:"transaction_id"`
	SourceWallet      string    `json:"source_wallet"`
	DestinationWallet string    `json:"destination_wallet"`
	Amount            float32   `json:"amount"`
	Description       string    `json:"transaction_description"`
	Date              time.Time `json:"date"`
}

func createIndex(rdb *redis.Client, ctx context.Context, indexName, fieldName, jsonPath string) error {
	errCreateIdx := rdb.Do(ctx, "FT.CREATE",
		indexName,
		"ON", "JSON", "SCHEMA", jsonPath,
		"AS", fieldName, "TEXT").Err()

	if errCreateIdx != nil {
		if strings.Compare(errCreateIdx.Error(), "Index already exists") == 0 {
			log.Printf("could not create index idx:description %s", errCreateIdx)
			return nil
		}
		return errCreateIdx
	}
	return nil
}

func Genesis(rdb *redis.Client, vaultWalletId string, startingBalance float32) error {
	ctx := context.Background()
	if vaultWalletId == "" || startingBalance <= 0 {
		return errors.New("could not process genesis with empty vaultId or <= 0 starting balance")
	}

	// creating search index on wallet Ids
	errCreateWalletIdIndex := createIndex(rdb, ctx,
		"idx:wallet:id",
		"wallet_id",
		"$.wallet_id")
	if errCreateWalletIdIndex != nil {
		return errCreateWalletIdIndex
	}

	// Creating search index on transaction description fields
	errCreateTransactionDescriptionIndex := createIndex(rdb, ctx,
		"idx:transaction_description",
		"transaction_description",
		"$.transaction_description")
	if errCreateTransactionDescriptionIndex != nil {
		return errCreateTransactionDescriptionIndex
	}

	// Test if vault account exists, exit
	searchVaultCmd := rdb.Do(ctx, "FT.SEARCH",
		"idx:wallet:id",
		fmt.Sprintf(`'@wallet_id:(%s)'`, vaultWalletId),
	)

	output, errGetVaultWallet := searchVaultCmd.Slice()

	if errGetVaultWallet != nil {
		log.Printf("error in command: %e for command : %v", errGetVaultWallet, searchVaultCmd.String())
		return errGetVaultWallet
	}

	if output[0] != 0 {
		log.Printf("vault account already exists, aborting")
		return errVaultAccountExist
	}

	newWallet := wallet{
		ID:      vaultWalletId,
		Balance: startingBalance,
	}
	strNewWallet, errStrNewWallet := json.Marshal(newWallet)
	if errStrNewWallet != nil {
		log.Printf("could not marshal vault wallet into json")
		return errStrNewWallet
	}
	errCreateVaultWallet := rdb.Do(ctx, "JSON.SET",
		fmt.Sprintf("wallet:%s", vaultWalletId),
		"$",
		strNewWallet,
	).Err()

	if errCreateVaultWallet != nil {
		log.Printf("could not create vault wallet %s with command", errCreateVaultWallet)
		return errCreateVaultWallet
	}

	log.Printf("successfully created vault wallet %s with starting balance %f",
		vaultWalletId,
		startingBalance)
	return nil
}

func postWallet(rdb *redis.Client) func(*gin.Context) {
	return func(c *gin.Context) {
		var receivedWallet wallet
		if errBind := c.BindJSON(&receivedWallet); errBind != nil {
			log.Printf("could not process received wallet, %s", errBind)
			return
		}
		log.Printf("received wallet:%+v", receivedWallet)

		c.IndentedJSON(http.StatusCreated, receivedWallet)
	}
}

func postTransaction(rdb *redis.Client) func(*gin.Context) {
	return func(c *gin.Context) {
		var receivedTransaction transaction
		if errBind := c.BindJSON(&receivedTransaction); errBind != nil {
			log.Printf("could not process received transaction, %s", errBind)
			return
		}
		log.Printf("received transaction:%+v", receivedTransaction)

		c.IndentedJSON(http.StatusCreated, receivedTransaction)
	}
}
