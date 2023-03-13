# Orakl Network Delegator

## Installation

```shell
yarn install
```

## Settings

Orakl Delegator requires to set the following environment variables.

- `DATABASE_URL`
- `APP_PORT`
- `CAVER_PRIVATE_KEY`
- `SIGNER_PRIVATE_KEY`

You can copy them from `.env.example` to `.env` and fill the appropriate values.

```shell
cp .env.example .env
```

## Local Development

```shell
brew install postgresql@14
brew services start postgresql@14
```

```shell
createdb orakl-delegator
#dropdb orakl-delegator
```

## Prisma

```shell
npx prisma format
npx prisma migrate dev --name init
```

## Running the app

```bash
# development
yarn run start

# watch mode
yarn run start:dev

# production mode
yarn run start:prod
```

## Test

```bash
# unit tests
yarn run test

# e2e tests
yarn run test:e2e

# test coverage
yarn run test:cov
```

## API swagger

```bash
http://localhost:3000/docs
```
