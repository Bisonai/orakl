#!/bin/bash
psql -h postgres -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" <<EOF
DELETE FROM vrf_keys CASCADE;
DELETE FROM listeners CASCADE;
DELETE FROM reporters CASCADE;
DELETE FROM chains CASCADE;
DELETE FROM services CASCADE;
EOF