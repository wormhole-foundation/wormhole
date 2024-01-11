# Wait for node to start
while ! /bin/netcat -z localhost 26657; do
  sleep 1
done

# Wait for first block
while [ $(curl localhost:26657/status -ks | jq ".result.sync_info.latest_block_height|tonumber") -lt 1 ]; do
  sleep 1
done

sleep 5

node deploy.js
