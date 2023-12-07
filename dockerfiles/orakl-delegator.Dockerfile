# node:20.10.0-slim
FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480 AS build

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY delegator delegator

RUN yarn delegator install

RUN yarn delegator build

FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480

WORKDIR /app

RUN apt-get update -y && apt-get install -y openssl

COPY --from=build /app/package.json /app/package.json

COPY --from=build /app/node_modules /app/node_modules

COPY --from=build /app/delegator /app/delegator

WORKDIR /app/delegator

CMD ["yarn", "start:prod"]