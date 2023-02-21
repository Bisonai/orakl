# Data Feed

## Locally launch 2 node operators submitting on single data feed

```shell
mkdir ~/.orakl-one ~/.orakl-two

# migrate
ORAKL_DIR=~/.orakl-one yarn cli migrate --force --migrationsPath src/cli/orakl-cli/migrations/
ORAKL_DIR=~/.orakl-two yarn cli migrate --force --migrationsPath src/cli/orakl-cli/migrations/

ORAKL_DIR=~/.orakl-one yarn cli adapter insert --chain localhost --file-path adapter/eth-usd.adapter.json
ORAKL_DIR=~/.orakl-two yarn cli adapter insert --chain localhost --file-path adapter/eth-usd.adapter.json

ORAKL_DIR=~/.orakl-one yarn cli aggregator insert --chain localhost --file-path aggregator/eth-usd.aggregator.json --adapter 0x7e6552824ce107ab0d6e4266ba6b93f0afe5aa576a491364fc01881a34ddb12b
ORAKL_DIR=~/.orakl-two yarn cli aggregator insert --chain localhost --file-path aggregator/eth-usd.aggregator.json --adapter 0x7e6552824ce107ab0d6e4266ba6b93f0afe5aa576a491364fc01881a34ddb12b

ORAKL_DIR=~/.orakl-two yarn cli kv update --chain localhost --key PUBLIC_KEY --value 0x90F79bf6EB2c4f870365E785982E1f101E93b906
ORAKL_DIR=~/.orakl-two yarn cli kv update --chain localhost --key PRIVATE_KEY --value 0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6

# flush
DEPLOYMENT_NAME=orakl-one yarn flush
DEPLOYMENT_NAME=orakl-two yarn flush

# launch orakl one
DEPLOYMENT_NAME=orakl-one ORAKL_DIR=~/.orakl-one yarn start:listener:aggregator
DEPLOYMENT_NAME=orakl-one ORAKL_DIR=~/.orakl-one yarn start:worker:aggregator
DEPLOYMENT_NAME=orakl-one ORAKL_DIR=~/.orakl-one yarn start:reporter:aggregator

# launch orakl two
DEPLOYMENT_NAME=orakl-two ORAKL_DIR=~/.orakl-two yarn start:listener:aggregator
DEPLOYMENT_NAME=orakl-two ORAKL_DIR=~/.orakl-two yarn start:worker:aggregator
DEPLOYMENT_NAME=orakl-two ORAKL_DIR=~/.orakl-two yarn start:reporter:aggregator
```
