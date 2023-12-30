# node:20.10.0-slim
FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480

RUN apt-get update && apt-get install -y curl jq

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY contracts contracts

COPY vrf vrf

RUN yarn contracts install

RUN yarn contracts compile

RUN mkdir -p /app/contracts/scripts/v0.1/tmp

WORKDIR /app/contracts