# Verifiable Random Function (VRF)

1. Generate VRF keys

To provide VRF service, one must generate VRF private and public keys.
All keys can be generated with the command below

```shell
yarn keygen
```

The output format of command is shown below

```shell
SK=
PK=
PK_X=
PK_Y=
KEY_HASH=
```

The generated `KEY_HASH` uniquely represents your VRF keys without exposing them.
This hash must be registered using `VRFCoordinator.registerProvingKey`.
Please share with Bisonai this `KEY_HASH` and your `PUBLIC_KEY` that corresponds to your wallet address.

2. Setup VRF keys

The keys generated in previous step are supplied to `yarn cli vrf insert` command.
Parameter `--chain` corresponds to the name network to which VRF keys will be associated.

```shell
yarn cli vrf insert \
    --chain ${chain} \
    --pk [PK] \
    --sk [SK] \
    --pk_x [PK_X] \
    --pk_y [PK_Y]
```

3. Setup listener

VRF node has to be able to catch on-chain requests.
You will need to know address of VRF coordinator (`vrfCoordinatorAddress`).
Emitted event is called `RandomWordsRequested`.

```
yarn cli listener insert \
    --service VRF \
    --chain ${chain} \
    --address ${vrfCoordinatorAddress} \
    --eventName RandomWordsRequested
```

4. Launch `listener`, `worker` and `reporter`

TODO update with Docker compose launch.

```shell
yarn start:listener:vrf
yarn start:worker:vrf
yarn start:reporter:vrf
```
