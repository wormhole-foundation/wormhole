BLOCK_JSON=$(curl -s -d '{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": "NSM5RDZDF7uxGWiUwhBqJcqCEw6g7axx4TxGYB7XZVt"}}'  -H "Content-Type: application/json" -X POST https://rpc.mainnet.near.org)
echo "$BLOCK_JSON" | jq > block.json

