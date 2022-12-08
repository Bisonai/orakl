## Contracts

### VRF

TODO
* change `s_consumers` to `s_nonce`?
* Fix `pendingRequestExists`
* Make `pendingRequestExists` more efficient (remove limitation on number of subcribers)
* `pendingRequestExists` is using `s_provingKeyHashes` in a strange way
* Update `getFeeTier` according to KLAY

* Subscription separation from VRF
** Apply KLAY to `cancelSubscriptionHelper`.
** Apply KLAY to `oracleWithDraw`

```
uint64 subId = createSubscription()

```

### Proving Keys

```
registerProvingKey
deregisterProvingKey
```

#### Transfer Ownership
```
requestSubscriptionOwnerTransfer
acceptSubscriptionOwnerTransfer
```

### Modify Subscription

```
createSubscription
cancelSubcription
```

#### Modify Consumers

```
addConsumer
removeConsumer
```
