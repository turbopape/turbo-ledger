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
    --data '{"wallet_id": "new_wallet", "tags": ["vip","rich"]}'
```

## Create a transaction
```shell
curl http://localhost:12000/transactions \
    --include \
    --header "Content-Type: application/json" \
    --request "POST" \
    --data '{"source_wallet": "src_wallet",
             "destination_wallet": "dst_wallet",
             "amount": 5,
             "transaction_description":"my cool transaction",
             "date":"2022-06-07T19:02:01.0Z"}'
```

## Search wallets by tags
```shell
curl http://localhost:12000/wallets\?q\=rich\
    --include \
    --header "Content-Type: application/json" \
    --request "GET" 
```