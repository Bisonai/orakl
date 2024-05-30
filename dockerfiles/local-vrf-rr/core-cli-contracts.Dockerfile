# node:20.10.0-slim
FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480 AS build

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package.json .

COPY yarn.lock .

COPY dockerfiles/local-vrf-rr/scripts/update-rr-migration.js update-rr-migration.js

COPY dockerfiles/local-vrf-rr/scripts/update-vrf-migration.js update-vrf-migration.js

COPY contracts/v0.1 contracts/v0.1

COPY vrf vrf

COPY util util

COPY core core

COPY cli cli

RUN yarn core install

RUN yarn core build

RUN yarn cli install

RUN yarn cli build

FROM node@sha256:18aacc7993a16f1d766c21e3bff922e830bcdc7b549bbb789ceb7374a6138480

WORKDIR /app

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

COPY --from=build /app/package.json /app/package.json

COPY --from=build /app/update-rr-migration.js /app/update-rr-migration.js

COPY --from=build /app/update-vrf-migration.js /app/update-vrf-migration.js

COPY --from=build /app/node_modules /app/node_modules

COPY --from=build /app/core/node_modules /app/core/node_modules

COPY --from=build /app/cli/node_modules /app/cli/node_modules

COPY --from=build /app/contracts/v0.1 /app/contracts/v0.1

COPY --from=build /app/vrf /app/vrf

COPY --from=build /app/util /app/util

COPY --from=build /app/core /app/core

COPY --from=build /app/cli /app/cli

CMD ["tail", "-f", "/dev/null"]