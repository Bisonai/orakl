# node:20.10.0-slim
FROM node@sha256:d480d6a3c334226e11064b2f34ac1a8846137e26a4f76e81ba7c63398759c384

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY contracts contracts

COPY vrf vrf

COPY fetcher fetcher

COPY core core

RUN yarn core install

RUN yarn core build