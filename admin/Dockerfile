FROM node:16.17.1-alpine3.15

RUN apk add --no-cache libc6-compat curl

WORKDIR /app

COPY . /app

COPY package.json /app

COPY yarn.lock /app

# To install modules with `git+https`: https://github.com/yarnpkg/yarn/issues/6881
RUN apk add git

# node-gyp ISSUE : https://github.com/AndreySenov/node-gyp-docker
RUN apk add --no-cache python3 make g++ && \
    yarn global add node-gyp@${VERSION} && \
    yarn cache clean && \
    node-gyp help && \
    mkdir $HOME/.cache && \
    chown -R node:node $HOME

RUN rm -rf /app/node_modules/ /app/yarn-error.log

# RUN yarn install
RUN yarn install --production

ENV NODE_ENV production

RUN yarn build

CMD [ "yarn", "start" ]
