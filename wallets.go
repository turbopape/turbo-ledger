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

var errWalletExists = errors.New("account exists")

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
	errCheckWalletExists := checkWalletExists(ctx, rdb, vaultWalletId)
	if errCheckWalletExists != nil {
		if errCheckWalletExists == errWalletExists {
			return nil
		}
		return errCheckWalletExists
	}

	newWallet := wallet{
		ID:      vaultWalletId,
		Balance: startingBalance,
	}
	return createWallet(ctx, rdb, vaultWalletId, startingBalance, newWallet)
}

func createWallet(ctx context.Context, rdb *redis.Client, walletId string, startingBalance float32, newWallet wallet) error {
	if walletId == "" {
		return errors.New("could not create wallet with empty wallet Id")
	}
	strNewWallet, errStrNewWallet := json.Marshal(newWallet)
	if errStrNewWallet != nil {
		log.Printf("could not marshal wallet into json")
		return errStrNewWallet
	}
	errCreateVaultWallet := rdb.Do(ctx, "JSON.SET",
		fmt.Sprintf("wallet:%s", walletId),
		"$",
		strNewWallet,
	).Err()

	if errCreateVaultWallet != nil {
		log.Printf("could not create wallet %s with command", errCreateVaultWallet)
		return errCreateVaultWallet
	}

	log.Printf("successfully created wallet %s with starting balance %f",
		walletId,
		startingBalance)
	return nil
}

func checkWalletExists(ctx context.Context, rdb *redis.Client, walletId string) error {
	if walletId == "" {
		return errors.New("could not check wallet with empty wallet Id")
	}

	searchWalletCmd := rdb.Do(ctx, "FT.SEARCH",
		"idx:wallet:id",
		fmt.Sprintf(`'@wallet_id:(%s)'`, walletId),
	)

	output, errGetWallet := searchWalletCmd.Slice()

	if errGetWallet != nil {
		log.Printf("error in command: %e for command : %v", errGetWallet, searchWalletCmd.String())
		return errGetWallet
	}

	if output[0].(int64) != 0 {
		log.Printf("Wallet already exists, aborting")
		return errWalletExists
	}

	return nil
}

func postWallet(rdb *redis.Client) func(*gin.Context) {
	return func(c *gin.Context) {
		var receivedWallet wallet
		if errBind := c.BindJSON(&receivedWallet); errBind != nil {
			log.Printf("could not process received wallet, %s", errBind)
			return
		}
		ctx := context.Background()
		walletId := receivedWallet.ID
		errCheckWalletExists := checkWalletExists(ctx, rdb, walletId)
		if errCheckWalletExists != nil {
			if errCheckWalletExists == errWalletExists {
				c.IndentedJSON(http.StatusConflict, receivedWallet)
				return
			}
			c.IndentedJSON(http.StatusInternalServerError, receivedWallet)
			return
		}

		errCreateWallet := createWallet(ctx, rdb, walletId, 0, receivedWallet)
		if errCreateWallet != nil {
			c.IndentedJSON(http.StatusInternalServerError, receivedWallet)
			return
		}

		c.IndentedJSON(http.StatusCreated, receivedWallet)
		return
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
