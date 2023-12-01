# node:18.12.1-slim
FROM node@sha256:0c3ea57b6c560f83120801e222691d9bd187c605605185810752a19225b5e4d9

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY cli cli

RUN yarn cli install --focus

RUN yarn cli build