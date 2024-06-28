# Monitoring scripts

## ECR + Helm Chart Check

```
./ecr-check.sh
```

```
orakl-core: count FAIL (7 tags)
(cypress) orakl-core: vrf FAIL (v0.3.1.20231103.0653.19ca105)
(cypress) orakl-core: request-response FAIL (v0.3.1.20231016.1000.749dac8)
(cypress) orakl-core: aggregator OK (v0.5.1.20240122.0131.44014e7)
(baobab) orakl-core: vrf OK (v0.5.1.20240111.1440.dacb2ef)
(baobab) orakl-core: vrf OK (v0.5.1.20240111.1440.dacb2ef)
(baobab) orakl-core: vrf OK (v0.5.1.20240111.1440.dacb2ef)
(baobab) orakl-core: request-response OK (v0.5.1.20240111.1440.dacb2ef)
(baobab) orakl-core: request-response OK (v0.5.1.20240122.0240.1672788)
(baobab) orakl-core: request-response OK (v0.5.1.20240111.1440.dacb2ef)
(baobab) orakl-core: aggregator OK (v0.5.1.20231221.0205.5ff1278)
(baobab) orakl-core: aggregator OK (v0.5.1.20240122.0240.1672788)
(baobab) orakl-core: aggregator OK (v0.5.1.20240122.0131.44014e7)
orakl-api: count OK
(cypress) orakl-api: api OK (v0.1.0.20231214.0553.3dd6e26)
(baobab) orakl-api: api OK (v0.1.0.20231214.0451.cb4bd3a)
orakl-cli: count OK
(cypress) orakl-cli: cli FAIL (v0.6.0.20230920.0750.99c4cdd)
(baobab) orakl-cli: cli OK (v0.6.0.20231229.0312.9433d53)
orakl-fetcher: count FAIL (6 tags)
(cypress) orakl-fetcher: fetcher OK (v0.0.1.20231221.0248.5ff1278)
(baobab) orakl-fetcher: fetcher OK (v0.0.1.20240122.0240.1672788)
orakl-delegator: count OK
(cypress) orakl-delegator: delegator FAIL (v0.0.1.20230707.0137.513e9f9)
(baobab) orakl-delegator: delegator OK (v0.0.1.20231211.0735.867d885)
```

## JSON RPC sync status check

```
./json-rpc-sync-check.sh
```

```
[INFO] Baobab json-rpc is synchronized
[INFO] Cypress json-rpc is synchronized
```
