#!/bin/sh

createdb orakl-test
DATABASE_URL="postgresql://${USER}@localhost:5432/orakl-test?schema=public" npx prisma migrate dev --name init
DATABASE_URL="postgresql://${USER}@localhost:5432/orakl-test?schema=public" yarn test
dropdb orakl-test
