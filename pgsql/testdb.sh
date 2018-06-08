#!/usr/bin/env bash

## This will setup Docker Postgres:Alpine container, run test and tear down Docker container.
## Make executable: $ chmod +x testdb.sh
## Run: $ ./testdb.sh
mkdir db

docker run --name wuid-postgres -e POSTGRES_PASSWORD=mysecretpassword -p 5432:5432 -d -v $(pwd)/db:/var/lib/postgresql/data postgres:alpine -c 'ssl=on'

sleep 30s

docker stop wuid-postgres

sleep 1s

cp certs/server.crt db/server.crt && cp certs/server.key db/server.key
chmod 0600 db/server.crt && chmod 0600 db/server.key
docker restart wuid-postgres 

sleep 3s

go test -cover -bench=.

docker kill wuid-postgres && docker rm wuid-postgres

rm -rf db