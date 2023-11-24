# node:18.12.1-alpine3.17
#FROM node@sha256:b375b98d1dcd56f5783efdd80a4d6ff5a0d6f3ce7921ec99c17851db6cba2a93

# node:18.12.1-slim
FROM node@sha256:0c3ea57b6c560f83120801e222691d9bd187c605605185810752a19225b5e4d9

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY tsconfig.json .

COPY src src

COPY prisma prisma

RUN yarn install

RUN yarn build

CMD ["yarn start:prod"]