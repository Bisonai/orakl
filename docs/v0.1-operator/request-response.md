# Request-Response

## 1. Setup listener

Request-Response node is listening on `RequestResponseCoordinator` smart contract.
When user requests data, `DataRequested` event is emmited.

```
yarn cli listener insert \
    --service RequestResponse \
    --chain ${chain} \
    --address ${coordinatorAddress} \
    --eventName DataRequested
```

## 2. Launch

TODO update with Docker compose launch

```shell
yarn start:listener:request_response
yarn start:worker:request_response
yarn start:reporter:request_response
```
