# Orakl Network Delegator

## Installation

```shell
yarn install
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

## API Link

```bash
http://localhost:3000/docs
```
