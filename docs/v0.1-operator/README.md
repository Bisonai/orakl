# v0.1 for operators

```
yarn add @bisonai-cic/icn-core
```

## Prerequisities

## Verifiable Random Function (VRF)

1. Generate VRF keys

To provide VRF service, one must generate VRF private and public keys.
All keys can be generated with the command below

```
yarn keygen
```

The output format of command is shown below

```
SK=
PK=
PK_X=
PK_Y=
KEY_HASH=
```

2. Setup VRF keys

The keys generated in previous step are supplied to `yarn cli vrf insert` command.
Parameter `--chain` corresponds to the name network to which VRF keys will be associated.

```
yarn cli vrf insert \
    --chain baobab \
    --pk [PK] \
    --sk [SK] \
    --pk_x [PK_X] \
    --pk_y [PK_Y]
```
