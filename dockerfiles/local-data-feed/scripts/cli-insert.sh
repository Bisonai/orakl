psql -h postgres -U ${POSTGRES_USER} -d ${POSTGRES_DB} <<EOF
SET search_path TO delegator;
INSERT INTO fee_payers ("privateKey") VALUES ('${DELEGATOR_REPORTER_PK}')
ON CONFLICT ("privateKey")
DO NOTHING
RETURNING *;
EOF

curl -s ${ORAKL_NETWORK_DELEGATOR_URL}/sign/initialize


yarn cli chain insert --name baobab_test
yarn cli service insert --name DATA_FEED
yarn cli delegator organizationInsert --name bisonai
yarn cli datafeed insert --source /app/tmp/updated_bulk.json
