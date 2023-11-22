# Orakl Network VRF

Orakl Network VRF is based on ECVRF that was proposed at [draft-irtf-cfrg-vrf-10](https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-vrf-10). Orakl Network VRF is used in [Orakl Network Core](https://github.com/Bisonai/orakl/tree/master/core) and [Orakl Network CLI](https://github.com/Bisonai/orakl/tree/master/cli).

## Basics

```
# prover
beta = VRF_hash(SK, alpha)
pi = VRF_prove(SK, alpha)

# verifier
beta = VRF_proof_to_hash(pi)
VRF_verify(PK, alpha, pi)

VRF_hash(SK, alpha) = VRF_proof_to_hash(VRF_prove(SK, alpha))
```

## Installation

```shell
yarn install
```

## Tests

```shell
yarn test
```

## Linter

```shell
yarn lint
```

## Publishing

`@bisonai/orakl-vrf` is published through [Github Actions pipeline](https://github.com/Bisonai/orakl/blob/master/.github/workflows/vrf.build+publish.yaml) when package version specified in `package.json` changes.

## Acknowledgements

Some parts of code were inspired by or copied from

- [node-ecrg](https://github.com/KenshiTech/node-ecvrf)
- [vrf-ts-256](https://github.com/cbrpunks/vrf-ts-256)
