# Orakl Network Delegator

## Installation

```shell
yarn install
```

## Settings

Orakl Delegator requires to set the following environment variables.

- `DATABASE_URL`
- `PROVIDER_URL`
- `APP_PORT`
- `DELEGATOR_FEEPAYER_PK`
- `TEST_DELEGATOR_REPORTER_PK` (necessary only to run test)

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
createdb orakl_delegator
#dropdb orakl_delegator
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

## Endpoints

### Health

- `GET http://localhost:3000/health`

## API swagger

- `http://localhost:3000/docs`

### List all signed transactions

- `GET http://localhost:3000/api/v1/sign`

### List all organization

- `GET http://localhost:3000/api/v1/organization`

### List all contract address

- `GET http://localhost:3000/api/v1/contract`

### List all function methods

- `GET http://localhost:3000/api/v1/function`

### List all reporter

- `GET http://localhost:3000/api/v1/reporter`

## Test

```bash
# unit tests
yarn run test

# e2e tests
yarn run test:e2e

# test coverage
yarn run test:cov
```
