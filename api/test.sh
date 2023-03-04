#!/bin/sh

DB_NAME=orakl-test
createdb ${DB_NAME}
DATABASE_URL="postgresql://${USER}@localhost:5432/${DB_NAME}?schema=public" npx prisma migrate dev --name init
DATABASE_URL="postgresql://${USER}@localhost:5432/${DB_NAME}?schema=public" yarn test
dropdb orakl-test
