# node:20.10.0-slim
FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480

RUN apt-get update && apt-get install -y curl jq

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY contracts/v0.1 contracts/v0.1

COPY vrf vrf

RUN yarn contracts-v01 install

RUN yarn contracts-v01 compile

RUN mkdir -p /app/contracts/v0.1/scripts/tmp

WORKDIR /app/contracts/v0.1