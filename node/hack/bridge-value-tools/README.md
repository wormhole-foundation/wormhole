# Bridge Value Tools
These tools help calculate Total Value Locked (TVL) and Total Value Minted (TVM) based on Global Accountant, Token Metadata DB, and Token Allowlist.

## How to run
```
cd accountant-dump
npm ci
npx ts-node main.ts > ../data/accountant.json

cd ../

# TODO: wget -O ./data/token-allowlist.json https://raw.githubusercontent.com/wormhole-foundation/wormhole-web-event-database/main/cloud_functions/token-allowlist-mainnet.json
# TODO: get token_metadata.json from token_metadata Cloud SQL table

go run ./
```