# turbo-ledger - A Scalable Ledger
This is an accompanying repo for the [Building turbo-ledger: A Scalable Ledger with Go and Redis](https://dev.to/turbopape/building-a-scalable-ledger-with-go-on-rediscloud-mo/stats) post.
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
    --data '{"wallet_id": "new_wallet", "tags": ["vip","rich"],"owner":"rafik}'
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

## Search wallets by owners
```shell
curl http://localhost:12000/wallets/owner?query\=bob\
    --include \
    --header "Content-Type: application/json" \
    --request "GET" 
```

## Search wallets by tags
```shell
curl http://localhost:12000/wallets/tags\?query\=rich\
    --include \
    --header "Content-Type: application/json" \
    --request "GET" 
```
