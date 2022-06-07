# WIP
# https://devnet.docs.injective.dev/cosmwasm-dapps/01_Cosmwasm_CW20_deployment_guide_%20on_%20Local.html#
# this script expects that you have injectived installed, ran setup.sh, and started a fresh chain
yes 12345678 | injectived tx wasm store artifacts/wormhole.wasm --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=json
# injectived query tx C93C7C6D3C4D35E2C223735D319957E47CF36D736E1A1898D1BC865F1A9F7DE3
yes 12345678 | injectived tx wasm instantiate 5 '{"gov_chain": 1, "gov_address": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=", "guardian_set_expirity": 86400, "initial_guardian_set": {"addresses": [{"bytes": "vvpCnVfNGLf4pNkaLamrSvBdD74="}], "expiration_time": 0}}' --label="Wormhole" --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --no-admin --output=json
# injectived query tx 5EF7786DC06B48BD0D72827248995614FDDA2DC9BD735A6E2569355BF7A3742D
# http://localhost:10337/swagger/#/Query/ContractsByCode
# curl -X GET "http://localhost:10337/cosmwasm/wasm/v1/code/5/contracts" -H "accept: application/json"
# {
#   "contracts": [
#     "inj1yvgh8xeju5dyr0zxlkvq09htvhjj20fnne37rc"
#   ],
#   "pagination": {
#     "next_key": null,
#     "total": "1"
#   }
# }
# CONTRACT=$(injectived query wasm list-contract-by-code $CODE_ID --output json | jq -r '.contracts[-1]')
injectived query wasm contract inj1yvgh8xeju5dyr0zxlkvq09htvhjj20fnne37rc --output json
injectived query wasm contract-state all inj1yvgh8xeju5dyr0zxlkvq09htvhjj20fnne37rc --output json
injectived query wasm contract-state smart inj1yvgh8xeju5dyr0zxlkvq09htvhjj20fnne37rc '{"guardian_set_info":{}}' --output json
yes 12345678 | injectived tx wasm execute inj1yvgh8xeju5dyr0zxlkvq09htvhjj20fnne37rc '{"post_message":{"message":"SGVsbG8sIENsYXJpY2U","nonce":69}}' --from genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000
yes 12345678 | injectived tx wasm store artifacts/cw20_wrapped_2.wasm --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --output=json
yes 12345678 | injectived tx wasm store artifacts/token_bridge_terra_2.wasm --from=genesis --chain-id="injective-1" --yes --fees=1500000000000000inj --gas=3000000 --output=json
yes 12345678 | injectived tx wasm instantiate 8 '{"gov_chain":1, "gov_address":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=", "wormhole_contract":"inj16jzpxp0e8550c9aht6q9svcux30vtyyyfagtwp", "wrapped_asset_code_id":7}' --label="tokenBridge" --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --no-admin
injectived query wasm list-contract-by-code 8 --output json | jq -r '.contracts[-1]'
yes 12345678 | injectived tx wasm execute inj124tapgv8wsn5t3rv2cvywh4ckkmj6mc6edtf5t '{ "deposit_tokens": {}}' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000 --amount 100000000000000000inj
injectived query bank balances inj124tapgv8wsn5t3rv2cvywh4ckkmj6mc6edtf5t
yes 12345678 | injectived tx wasm execute inj124tapgv8wsn5t3rv2cvywh4ckkmj6mc6edtf5t '{"initiate_transfer":{"asset":{"amount":"1000000000000000", "info":{"native_token":{"denom":"inj"}}},"recipient_chain":2,"recipient":"AAAAAAAAAAAAAAAAQgaUIGlCBpQgaUIGlCBpQgaUIGk=","fee":"1000000","nonce":69}}' --from=genesis --chain-id="injective-1" --yes --fees=1000000000000000inj --gas=2000000
