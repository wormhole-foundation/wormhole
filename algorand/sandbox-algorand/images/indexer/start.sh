#!/bin/bash

# Start indexer daemon. There are various configurations controlled by
# environment variables.
#
# Configuration:
#   DISABLED          - If set start a server that returns an error instead of indexer.
#   CONNECTION_STRING - the postgres connection string to use.
#   SNAPSHOT          - snapshot to import, if set don't connect to algod.
#   PORT              - port to start indexer on.
#   ALGOD_ADDR        - host:port to connect to for algod.
#   ALGOD_TOKEN       - token to use when connecting to algod.


export PORT="8980"
export CONNECTION_STRING="host=localhost port=5432 user=algorand password=algorand dbname=indexer_db sslmode=disable"
export ALGOD_ADDR="localhost:4001"
export ALGOD_TOKEN="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

set -e
set -x
start_with_algod() {
  echo "Starting indexer against algod."

  for i in 1 2 3 4 5; do
    wget "${ALGOD_ADDR}"/genesis -O genesis.json && break
    echo "Algod not responding... waiting."
    sleep 15
  done

  if [ ! -f genesis.json ]; then
    echo "Failed to create genesis file!"
    exit 1
  fi

#  PGPASSWORD=algorand psql --host=algo-indexer-db --port=5432 --username=algorand --dbname=indexer_db -c "DROP DATABASE IF EXISTS postgres"
#  PGPASSWORD=algorand psql --host=algo-indexer-db --port=5432 --username=algorand --dbname=indexer_db -c "DROP DATABASE IF EXISTS template0"
#  PGPASSWORD=algorand psql --host=algo-indexer-db --port=5432 --username=algorand --dbname=indexer_db -c "DROP DATABASE IF EXISTS template1"
#  PGPASSWORD=algorand psql --host=algo-indexer-db --port=5432 --username=algorand -c "DROP DATABASE IF EXISTS indexer_db"
#  PGPASSWORD=algorand psql --host=algo-indexer-db --port=5432 --username=algorand -c "CREATE DATABASE indexer_db"

  /tmp/algorand-indexer daemon \
    --dev-mode \
    --server ":$PORT" \
    --data-dir "/opt/data" \
    --enable-all-parameters \
    -P "$CONNECTION_STRING" \
    --algod-net "${ALGOD_ADDR}" \
    --algod-token "${ALGOD_TOKEN}" \
    --genesis "genesis.json" \
    --logfile "/dev/stdout" >> /tmp/command.txt
}

import_and_start_readonly() {
  echo "Starting indexer with DB."

  # Extract the correct dataset
  ls -lh  /tmp
  mkdir -p /tmp/indexer-snapshot
  echo "Extracting ${SNAPSHOT}"
  tar -xf "${SNAPSHOT}" -C /tmp/indexer-snapshot

  /tmp/algorand-indexer import \
    -P "$CONNECTION_STRING" \
    --genesis "/tmp/indexer-snapshot/algod/genesis.json" \
    /tmp/indexer-snapshot/blocktars/* \
    --logfile "/tmp/indexer-log.txt" >> /tmp/command.txt

  /tmp/algorand-indexer daemon \
    --dev-mode \
    --server ":$PORT" \
    -P "$CONNECTION_STRING" \
    --logfile "/tmp/indexer-log.txt" >> /tmp/command.txt
}

disabled() {
  go run /tmp/disabled.go -port "$PORT" -code 400 -message "Indexer disabled for this configuration."
}

if [ ! -z "$DISABLED" ]; then
  disabled
elif [ -z "${SNAPSHOT}" ]; then
  start_with_algod
else
  import_and_start_readonly
fi

sleep infinity
