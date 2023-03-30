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
- `DELEGATOR_REPORTER_PK`

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

## Endpoints

App

- `GET /health`
- `GET /api`

Sign

- `GET /api/v1/sign/{tx}`
- `POST /api/v1/sign/{tx}`

Organization

- `POST /api/v1/organization/{name}`
- `GET /api/v1/organization/{id}`
- `DELETE /api/v1/organization/{id}`

Contract

- `POST /api/v1/contract/{address}`
- `GET /api/v1/contract/{id}`
- `DELETE /api/v1/contract/{id}`

Function

- `POST /api/v1/function/{functionName}`
- `GET /api/v1/function/{id}`
- `DELETE /api/v1/function/{id}`

Reporter

- `POST /api/v1/reporter/{address}`
- `GET /api/v1/reporter/{id}`
- `DELETE /api/v1/reporter/{id}`

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
