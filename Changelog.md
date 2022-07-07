# Change Log
## [naive-version] - 2022-07-08
## Added
- initial structure
- gitignore
- Readme
- Changelog
- Dockerfile
- LICENSE
- cli env vars + dot env
- redis connection
- gin router + empty first function
- skeletons postwallet and posttransaction as closure over rdb
- Implemented genesis function
- Implemented create wallet if not exists
- Implemented post transactions, verify balance then execute and update balance, using optimistic log  - not thread safe
- Implemented search wallets by tags