FROM node:20-bookworm

WORKDIR /app
COPY contracts/v0.1/hardhat.config.cjs /app/hardhat.config.cjs
COPY contracts/v0.1/package.json /app/package.json 
RUN yarn