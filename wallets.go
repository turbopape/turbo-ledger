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

	"sync"
)

var m sync.Mutex
var errWalletExists = errors.New("wallet already exists")
var errWalletNotFound = errors.New("wallet not found")
var errTransactionWrongAmount = errors.New("wrong transaction amount")
var errTransactionNotEnoughBalance = errors.New("not enough balance")
var errBalanceChanged = errors.New("balance changed by other client")

type Wallet struct {
	ID           string         `json:"wallet_id"`
	Transactions []*Transaction `json:"transactions"`
	Balance      float32        `json:"balance"`
	Tags         []string       `json:"tags"`
}

type Transaction struct {
	ID                string    `json:"transaction_id"`
	SourceWallet      string    `json:"source_wallet"`
	DestinationWallet string    `json:"destination_wallet"`
	Amount            float32   `json:"amount"`
	Description       string    `json:"transaction_description"`
	Date              time.Time `json:"date"`
}

func createIndex(rdb *redis.Client, ctx context.Context, indexName, fieldName, jsonPath, schemaType string) error {
	errCreateIdx := rdb.Do(ctx, "FT.CREATE",
		indexName,
		"ON", "JSON",
		"SCHEMA", jsonPath,
		"AS", fieldName, schemaType).Err()

	if errCreateIdx != nil {
		if strings.Compare(errCreateIdx.Error(), "Index already exists") == 0 {
			log.Printf("could not create index idx:description %s", errCreateIdx)
			return nil
		}
		return errCreateIdx
	}
	return nil
}

// Genesis
func Genesis(rdb *redis.Client, vaultWalletId string, startingBalance float32) error {
	ctx := context.Background()
	if vaultWalletId == "" || startingBalance <= 0 {
		return errors.New("could not process genesis with empty vaultId or <= 0 starting balance")
	}

	// creating search index on Wallet Ids
	errCreateWalletIdIndex := createIndex(rdb, ctx,
		"idx:wallet:id",
		"wallet_id",
		"$.wallet_id",
		"TEXT")
	if errCreateWalletIdIndex != nil {
		return errCreateWalletIdIndex
	}

	// Creating search index on Transaction description fields
	errCreateTransactionDescriptionIndex := createIndex(rdb, ctx,
		"idx:wallet:tags",
		"tags",
		"$.tags.*",
		"TAG")
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

	newWallet := Wallet{
		ID:      vaultWalletId,
		Balance: startingBalance,
	}
	return createWallet(ctx, rdb, vaultWalletId, startingBalance, newWallet)
}

// Wallets
func createWallet(ctx context.Context, rdb *redis.Client, walletId string, startingBalance float32, newWallet Wallet) error {
	if walletId == "" {
		return errors.New("could not create Wallet with empty Wallet Id")
	}
	strNewWallet, errStrNewWallet := json.Marshal(newWallet)
	if errStrNewWallet != nil {
		log.Printf("could not marshal Wallet into json")
		return errStrNewWallet
	}
	errCreateVaultWallet := rdb.Do(ctx, "JSON.SET",
		fmt.Sprintf("wallet:%s", walletId),
		"$",
		strNewWallet,
	).Err()

	if errCreateVaultWallet != nil {
		log.Printf("could not create Wallet %s with command", errCreateVaultWallet)
		return errCreateVaultWallet
	}

	log.Printf("successfully created Wallet %s with starting balance %f",
		walletId,
		startingBalance)
	return nil
}

func checkWalletExists(ctx context.Context, rdb *redis.Client, walletId string) error {
	if walletId == "" {
		return errors.New("could not check Wallet with empty Wallet Id")
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
		var receivedWallet Wallet
		if errBind := c.BindJSON(&receivedWallet); errBind != nil {
			log.Printf("could not process received Wallet, %s", errBind)
			return
		}
		receivedWallet.Transactions = make([]*Transaction, 0)
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

func getWalletBalance(ctx context.Context, rdb *redis.Client, walletID string) (float32, error) {
	getBalanceCmd := rdb.Do(ctx, "JSON.GET",
		"wallet:"+walletID,
		"$.balance",
	)

	rawOutput, errGetBalanceCmd := getBalanceCmd.Text()
	var output []interface{}
	json.Unmarshal([]byte(rawOutput), &output)

	if errGetBalanceCmd != nil {
		log.Printf("error %e for command : %v", errGetBalanceCmd, getBalanceCmd.String())
		return 0, errGetBalanceCmd
	}

	return float32(output[0].(float64)), nil
}

// Transactions
func ProcessTransaction(ctx context.Context, rdb *redis.Client, transaction Transaction, maxAttempts int) error {
	if transaction.Amount <= 0 {
		return errTransactionWrongAmount
	}

	//Acquire single thread lock
	m.Lock()

	//Make sure to unlock on exiting transaction
	defer m.Unlock()

	//Getting source balance, discarding if wallet not found or if not enough balance
	srcBalance, errGetSrcBalacnce := getWalletBalance(ctx, rdb, transaction.SourceWallet)
	if errGetSrcBalacnce != nil {
		return errGetSrcBalacnce
	}
	if transaction.Amount > srcBalance {
		log.Printf("Not enough balance for wallet %s, required %f but has %f", transaction.SourceWallet, transaction.Amount, srcBalance)
		return errTransactionNotEnoughBalance
	}

	//Getting destination Wallet balance, discarding if wallet not found
	_, errGetDstBalance := getWalletBalance(ctx, rdb, transaction.DestinationWallet)
	if errGetDstBalance != nil {
		return errGetDstBalance
	}

	// watch src and dst balance, emit transaction to write transaction in src, mirror transaction in dst,
	// and update respective balances

	if err := rdb.Do(ctx, "WATCH", "wallet:"+transaction.SourceWallet).Err(); err != nil {
		log.Printf("could not watch on source wallet %s", err)
		return err
	}
	if err := rdb.Do(ctx, "WATCH", "wallet:"+transaction.DestinationWallet).Err(); err != nil {
		log.Printf("could not watch on dest wallet %s", err)
		return err
	}
	if err := rdb.Do(ctx, "MULTI").Err(); err != nil {
		log.Printf("could not start transaction %s", err)
		return err
	}
	errAttemptTransaction := attemptTransaction(rdb, transaction)
	for i := 1; errAttemptTransaction != nil && i < maxAttempts; i++ {
		log.Printf("got error process transaction; processing %d / maxAttempts", i+1)
		errAttemptTransaction = attemptTransaction(rdb, transaction)
	}

	return errAttemptTransaction
}

func addTransactionToWallet(ctx context.Context, rdb *redis.Client, walletID string, transaction Transaction, inverted bool) error {

	if !inverted {
		transaction.Amount = -transaction.Amount
	}

	transactionJson, errTransactionJson := json.Marshal(transaction)
	if errTransactionJson != nil {
		log.Printf("could not marshal transaction %v %s", transactionJson, errTransactionJson)
		return errTransactionJson
	}
	errAddTransactionToWallet := rdb.Do(ctx, "JSON.ARRAPPEND",
		"wallet:"+walletID,
		"$.transactions",
		transactionJson,
	).Err()

	if errAddTransactionToWallet != nil {
		log.Printf("could not process transction : %v %s", transaction, errAddTransactionToWallet)
		rdb.Do(ctx, "DISCARD")
		return errAddTransactionToWallet
	}

	errUpdateBalance := rdb.Do(ctx, "JSON.NUMINCRBY",
		"wallet:"+walletID,
		"$.balance",
		transaction.Amount,
	).Err()

	if errUpdateBalance != nil {
		log.Printf("could not update amount for wallet: %s %s", walletID, errAddTransactionToWallet)
		rdb.Do(ctx, "DISCARD")
		return errAddTransactionToWallet
	}

	return nil
}

func attemptTransaction(rdb *redis.Client, transaction Transaction) error {
	ctx := context.Background()
	// watching wallets balances

	// writing
	errAddTransactionToSrc := addTransactionToWallet(ctx, rdb, transaction.SourceWallet, transaction, false)
	if errAddTransactionToSrc != nil {
		if err := rdb.Do(ctx, "DISCARD").Err(); err != nil {
			log.Printf("could not discard transaction")
		}
		return errAddTransactionToSrc
	}
	errAddTransactionToDst := addTransactionToWallet(ctx, rdb, transaction.DestinationWallet, transaction, true)
	if errAddTransactionToDst != nil {
		if err := rdb.Do(ctx, "DISCARD").Err(); err != nil {
			log.Printf("could not discard transaction")
		}
		return errAddTransactionToDst
	}

	return rdb.Do(ctx, "EXEC").Err()

}

func postTransaction(rdb *redis.Client) func(*gin.Context) {
	return func(c *gin.Context) {
		ctx := context.Background()
		var receivedTransaction Transaction
		if errBind := c.BindJSON(&receivedTransaction); errBind != nil {
			log.Printf("could not process received Transaction, %s", errBind)
			return
		}
		log.Printf("received Transaction:%+v", receivedTransaction)

		if errProcessTransaction := ProcessTransaction(ctx, rdb, receivedTransaction, 3); errProcessTransaction != nil {
			c.IndentedJSON(http.StatusInternalServerError, receivedTransaction)
			return
		}

		c.IndentedJSON(http.StatusCreated, receivedTransaction)
	}
}

func searchWalletsByTags(rdb *redis.Client) func(*gin.Context) {
	return func(c *gin.Context) {
		ctx := context.Background()
		if query, ok := c.GetQuery("query"); ok {
			searchTransactionCmd := rdb.Do(ctx, "FT.SEARCH",
				"idx:wallet:tags",
				fmt.Sprintf(`'@tags:{%s}'`, query),
			)
			output, errDescriptionSearch := searchTransactionCmd.Slice()
			if errDescriptionSearch != nil {
				log.Printf("could not search with query %s command %s", query, searchTransactionCmd.String())
				c.Status(http.StatusInternalServerError)
				return
			}
			if output[0].(int64) > 0 {
				c.IndentedJSON(http.StatusFound, map[string]interface{}{"wallet_id": output[1], "data": output[2]})
				return
			}
		}
		c.Status(http.StatusNotFound)

	}
}
