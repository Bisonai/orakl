# node:20.10.0-slim
FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480 AS build

ARG SERVICE

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY dockerfiles/local-vrf-rr/scripts/update-rr-migration.js update-rr-migration.js

COPY dockerfiles/local-vrf-rr/scripts/update-vrf-migration.js update-vrf-migration.js

COPY dockerfiles/local-vrf-rr/scripts/update-hardhat-network.js update-hardhat-network.js

COPY dockerfiles/local-vrf-rr/envs/vrf-keys-localhost.json vrf-keys-localhost.json 
COPY dockerfiles/local-vrf-rr/envs/vrf-keys-baobab.json vrf-keys-baobab.json 
COPY dockerfiles/local-vrf-rr/envs/vrf-keys-cypress.json vrf-keys-cypress.json 

COPY contracts/v0.1 contracts/v0.1

COPY vrf vrf

COPY util util

COPY core core

COPY cli cli

COPY dockerfiles/local-vrf-rr/scripts/setup-${SERVICE}.sh setup.sh
RUN chmod +x setup.sh

RUN yarn core install \
    && yarn core build \
    && yarn cli install \
    && yarn cli build

FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480

WORKDIR /app

RUN apt-get update \
    && apt-get install -y curl postgresql-client netcat-openbsd jq --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /app/package.json /app/package.json

COPY --from=build /app/update-rr-migration.js /app/update-rr-migration.js

COPY --from=build /app/update-vrf-migration.js /app/update-vrf-migration.js

COPY --from=build /app/update-hardhat-network.js /app/update-hardhat-network.js

COPY --from=build /app/vrf-keys-localhost.json /app/vrf-keys-localhost.json
COPY --from=build /app/vrf-keys-baobab.json /app/vrf-keys-baobab.json
COPY --from=build /app/vrf-keys-cypress.json /app/vrf-keys-cypress.json

COPY --from=build /app/node_modules /app/node_modules

COPY --from=build /app/core/node_modules /app/core/node_modules

COPY --from=build /app/cli/node_modules /app/cli/node_modules

COPY --from=build /app/contracts/v0.1 /app/contracts/v0.1

COPY --from=build /app/vrf /app/vrf

COPY --from=build /app/util /app/util

COPY --from=build /app/core /app/core

COPY --from=build /app/cli /app/cli

COPY --from=build /app/setup.sh /app/setup.sh

RUN chmod +x setup.sh

CMD [ "/app/setup.sh" ]
