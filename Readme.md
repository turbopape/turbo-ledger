# turbo-ledger - A Scalable Ledger
## Building and Running API
```shell
go get .
go run .
```

## Create a wallet
```shell
curl http://localhost:12000/wallets \
    --include \
    --header "Content-Type: application/json" \
    --request "POST" \
    --data '{"wallet_id": "new_wallet"}'
```

## Create a transaction
```shell
curl http://localhost:12000/transactions \
    --include \
    --header "Content-Type: application/json" \
    --request "POST" \
    --data '{"source_wallet": "source_wallet",
             "destination_wallet": "destination_wallet",
             "amount": 49.99,
             "date":"2022-06-07T19:02:01.0Z"}
```