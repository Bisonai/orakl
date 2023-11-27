# node:18.16.1-slim
FROM node@sha256:c20aa55f263e79547acc1c071f6186fd54cdeb0da31d717ccd10c5f7de843300

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY package.json .

COPY tsconfig.json .

COPY src src

COPY migrations migrations

RUN yarn install && yarn build

CMD ["yarn", "start"]
