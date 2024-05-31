# Run VRF and RR services locally

From the root of the repository, execute the following command:

```bash
SERVICE=rr docker-compose -f docker-compose.local-core.yaml up -d
```

Replace `SERVICE` with whichever service you'd like to run. The options are `vrf` and `rr` which represent VRF and Request-Response services respectively. To ensure your docker images that are used in the `docker-compose.local-core.yaml` file are up to date, you can add `--build` flag to the above command, which will rebuild all

Here is what happens after the above command is run:

- `api`, `postgres`, `redis`, and `json-rpc` services will start as separate docker containers
- `postgres` will get populated with necessary data:
  - chains
  - services
  - vrf keys in case if the service is vrf
  - listener (after contracts are deployed)
  - reporter (after contracts are deployed)
- migration files in `contracts/v0.1/migration/` get updated with generated keys and other values
- relevant coordinator and prepayment contracts get deployed

Keep in mind that you'll need the `keyHash` value for VRF consumer and update it in `vrf-consumer/scripts/utils.ts`

You can spin up the listener, worker, and reporter services from [core](../../core/) and make requests to VRF or Request-Response consumers after deploying consumer contracts.

### Notes

- The current automation is not designed to run both VRF and Request-Response services.
- Therefore, you'll need to run one service (either VRF or Request-Response) at a time and make sure to restart the `json-rpc` docker service if you'd like to run a different service (VRF or Request-Response).
