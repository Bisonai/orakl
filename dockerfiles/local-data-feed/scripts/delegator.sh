#!/bin/bash

npx prisma migrate deploy

psql -h postgres -U df -d test <<EOF
SET search_path TO delegator;
INSERT INTO fee_payers ("privateKey") VALUES ('${DELEGATOR_REPORTER_PK}')
ON CONFLICT ("privateKey")
DO NOTHING
RETURNING *;
EOF

echo "fee payer has been inserted"

yarn start:prod