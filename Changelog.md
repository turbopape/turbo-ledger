# Change Log
## [0.1.0] - 2022-07-09
### Removed
- idx:wallet:id
### Changed
- don't use FT.SEARCH to check for wallet's existence
- refactor searchWallet to be generic
- search has paths in api /wallet/tags, /wallet/owner
### Added
- wallet owner, idx:wallet:owner index
- serach on wallet owner
### Fixed
- in api when query is bad, return 400 malformed query
- titles formatting in Changelog

## [minor-additions] - 2022-07-09
### Added
- uuid to generate transaction Id
## [distributed-redlock-scalable-version] - 2022-07-08
### Removed
- Single thread locking / unlocking
### Added
- Distributed locking using redis - redlock

## [single-instance-thread-safe-version] - 2022-07-08
### Added
- Use single thread locking / unlocking to prevent race conditions on balance
- 
## [naive-version] - 2022-07-08
### Added
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