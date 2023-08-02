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

ARG ORAKL_URL
ARG MONITOR_URL
ARG DELEGATOR_URL

# This value must be external ip of the pods
# Because the nextjs app is running in the browser, it needs to be able to access the api from client side
ENV NEXT_PUBLIC_API_BASE_URL=${ORAKL_URL} 
ENV NEXT_PUBLIC_API_QUEUES_URL=${MONITOR_URL}
ENV NEXT_PUBLIC_API_DELEGATOR_URL=${DELEGATOR_URL}

RUN yarn build

CMD ["yarn", "start"]
