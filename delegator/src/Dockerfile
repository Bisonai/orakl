# node:18.12.1-alpine3.17
#FROM node@sha256:b375b98d1dcd56f5783efdd80a4d6ff5a0d6f3ce7921ec99c17851db6cba2a93

# node:18.16.1-slim
FROM node@sha256:c20aa55f263e79547acc1c071f6186fd54cdeb0da31d717ccd10c5f7de843300

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY package.json .

COPY tsconfig.json .

COPY src src

COPY prisma prisma

RUN yarn install

RUN yarn build

CMD ["yarn start:prod"]