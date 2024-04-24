# node:20.10.0-slim
FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY cli cli

COPY vrf vrf

RUN yarn cli install

RUN yarn cli build

WORKDIR /app/cli