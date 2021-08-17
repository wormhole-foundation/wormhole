# Wait for node to start
while ! /bin/netcat -z localhost 26657; do
  sleep 1
done
# Wait for first block
while [ $(curl localhost:26657/status -ks | jq ".result.sync_info.latest_block_height|tonumber") -lt 1 ]; do
  sleep 1
done

sleep 2

python deploy.py "http://terra-lcd:1317" "columbus-4" "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius" 1 "0000000000000000000000000000000000000000000000000000000000000004" beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe 86400
echo "Going to sleep, interrupt if running manually"
sleep infinity
