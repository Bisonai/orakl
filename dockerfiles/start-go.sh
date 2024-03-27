#!/bin/sh

_DATABASE_URL="${DATABASE_URL}"
if echo "$_DATABASE_URL" | grep -q "\?"; then
    _DATABASE_URL="${_DATABASE_URL}&sslmode=disable"
else
    _DATABASE_URL="${_DATABASE_URL}?sslmode=disable"
fi

migrate -database "$_DATABASE_URL" -verbose -path ./migrations up || exit 1
$1 || exit 1