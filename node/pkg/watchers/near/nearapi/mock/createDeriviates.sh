# success_mod1/: all fields of last_ds_final_block and last_final_block set to A5mwZmMzNZM39BVuEVfupMrEpvuCuRt6u9kJ1JGupgkx
find "$(dirname "$(realpath "$0")")/success_mod1/" -type f -name '*.json' -print0 | xargs -r -0 sed -i -e 's/"last_ds_final_block": ".*"/"last_ds_final_block": "A5mwZmMzNZM39BVuEVfupMrEpvuCuRt6u9kJ1JGupgkx"/g'
find "$(dirname "$(realpath "$0")")/success_mod1/" -type f -name '*.json' -print0 | xargs -r -0 sed -i -e 's/"last_final_block": ".*"/"last_final_block": "A5mwZmMzNZM39BVuEVfupMrEpvuCuRt6u9kJ1JGupgkx"/g'


# Generate some synthetic data
cp -r success/ synthetic


# 3Ms2KQ3gVeNa7Zkm8bt26: e60331f939353cf772ac9ad0d5b9ecd954ce2cb81d684d97bb3950c4fcdaac1c.json
read -r -d '' TX0_wrong_block << EOM
{
    "result": {
      "receipts_outcome": [
        {
          "block_hash": "A5mwZmMzNZM39BVuEVfupMrEpvuCuRt6u9kJ1JGupgkx",
          "outcome": {
            "executor_id": "contract.wormhole_crypto.near",
            "logs": [
              "wormhole/src/lib.rs#412: publish_message  prepaid_gas: \"94217765723035\"   used_gas: \"561737236755\"  delta: \"93653874829714\"",
              "EVENT_JSON:{\"standard\":\"wormhole\",\"event\":\"publish\",\"data\":\"0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":76538233,\"emitter\":\"148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7\",\"seq\":261,\"block\":76538234}"
            ],
            "status": {
              "SuccessValue": "MjYx"
            }
          }
        }
      ]
    }
  }
EOM

# 4VuUfFU7DUjrv15PdwknXZohW: 9ff5b7d38d730abec5fe887e0685b7b53e9edbb98ed504066f7c58a87a2ce97b.json
read -r -d '' TX0_wrong_sequence << EOM
{
    "result": {
      "receipts_outcome": [
        {
          "block_hash": "6zPnFkHojNQpbRgALHgRnbzhFvp55hido4Gv645nR8zf",
          "outcome": {
            "executor_id": "contract.wormhole_crypto.near",
            "logs": [
              "wormhole/src/lib.rs#412: publish_message  prepaid_gas: \"94217765723035\"   used_gas: \"561737236755\"  delta: \"93653874829714\"",
              "EVENT_JSON:{\"standard\":\"wormhole\",\"event\":\"publish\",\"data\":\"0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":76538233,\"emitter\":\"148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7\",\"seq\":262,\"block\":76538234}"
            ],
            "status": {
              "SuccessValue": "MjYx"
            }
          }
        }
      ]
    }
  }
EOM

# VLAU: 69641fb93ec440861372eaad83ce4452e449e91b62a00b3acc3b3367a5df88cd.json
read -r -d '' TX1 << EOM
{
    "result": {
      "receipts_outcome": [
        {
          "block_hash": "9AEuLtXe4JgJGnwY6ZZE6PmkPcEYpQqqUzwDMzUsMgBT",
          "outcome": {
            "executor_id": "contract.wormhole_crypto.near",
            "logs": [
              "wormhole/src/lib.rs#412: publish_message  prepaid_gas: \"94217765723035\"   used_gas: \"561737236755\"  delta: \"93653874829714\"",
              "EVENT_JSON:{\"standard\":\"wormhole\",\"event\":\"publish\",\"data\":\"0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":76538233,\"emitter\":\"148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7\",\"seq\":261,\"block\":76538230}"
            ],
            "status": {
              "SuccessValue": "MjYx"
            }
          }
        }
      ]
    }
  }
EOM

# VLAV: 2042fd1551c4700259fc48a5e0f476aae439c15e0f61250aef1a27f5ca987a26.json
read -r -d '' TX2 << EOM
{
    "result": {
      "receipts_outcome": [
        {
          "block_hash": "G3r7EszAnX2ecbV4jX8e7Ls9vamrwHnn19UP4SeUL5qv",
          "outcome": {
            "executor_id": "contract.wormhole_crypto.near",
            "logs": [
              "wormhole/src/lib.rs#412: publish_message  prepaid_gas: \"94217765723035\"   used_gas: \"561737236755\"  delta: \"93653874829714\"",
              "EVENT_JSON:{\"standard\":\"wormhole\",\"event\":\"publish\",\"data\":\"0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":76538233,\"emitter\":\"148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7\",\"seq\":262,\"block\":76538232}"
            ],
            "status": {
              "SuccessValue": "MjYy"
            }
          }
        }
      ]
    }
  }
EOM

# VLAW: bf03637a96b3a19be95cc218c1fc74a7b5eb8abb0a385d37dbe2bd659903e253.json
read -r -d '' TX3 << EOM
{
    "result": {
      "receipts_outcome": [
        {
          "block_hash": "6eCgeVSC4Hwm8tAVy4qNQpnLs4S9EpzRjGtAipwZ632A",
          "outcome": {
            "executor_id": "contract.wormhole_crypto.near",
            "logs": [
              "wormhole/src/lib.rs#412: publish_message  prepaid_gas: \"94217765723035\"   used_gas: \"561737236755\"  delta: \"93653874829714\"",
              "EVENT_JSON:{\"standard\":\"wormhole\",\"event\":\"publish\",\"data\":\"0100000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000000f0108bc32f7de18a5f6e1e7d6ee7aff9f5fc858d0d87ac0da94dd8d2a5d267d6b00160000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":76538233,\"emitter\":\"148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7\",\"seq\":263,\"block\":76538236}"
            ],
            "status": {
              "SuccessValue": "MjYz"
            }
          }
        }
      ]
    }
  }
EOM


base53="6gFEzydsKaV1nnsuc5E6afydzayzn9siWm332NqAAod8"
hash=$(printf '{"id": "dontcare", "jsonrpc": "2.0", "method": "tx", "params": ["'"$base53"'", "contract.wormhole_crypto.near"]}' | sha256sum | head -c 64)
filename="$(dirname "$(realpath "$0")")/synthetic/$hash.json"
echo "$TX0_wrong_block" > "$filename"

base53="6gFEzydsKaV1nnxWnb9wAaBXRQ92Zk4v99MBxr31WWpW"
hash=$(printf '{"id": "dontcare", "jsonrpc": "2.0", "method": "tx", "params": ["'"$base53"'", "contract.wormhole_crypto.near"]}' | sha256sum | head -c 64)
filename="$(dirname "$(realpath "$0")")/synthetic/$hash.json"
echo "$TX0_wrong_sequence" > "$filename"

base53="7RJ5bcEyBLDXFbmDSNzHeAKUC5z7z3Du5mKLY7FuuwoE"
hash=$(printf '{"id": "dontcare", "jsonrpc": "2.0", "method": "tx", "params": ["'"$base53"'", "contract.wormhole_crypto.near"]}' | sha256sum | head -c 64)
filename="$(dirname "$(realpath "$0")")/synthetic/$hash.json"
echo "$TX1" > "$filename"

base53="7RJ5bcEyBLDXFbmDSNzHeAKUC5z7z3Du5mKLY7FuuwoF"
hash=$(printf '{"id": "dontcare", "jsonrpc": "2.0", "method": "tx", "params": ["'"$base53"'", "contract.wormhole_crypto.near"]}' | sha256sum | head -c 64)
filename="$(dirname "$(realpath "$0")")/synthetic/$hash.json"
echo "$TX2" > "$filename"

base53="7RJ5bcEyBLDXFbmDSNzHeAKUC5z7z3Du5mKLY7FuuwoG"
hash=$(printf '{"id": "dontcare", "jsonrpc": "2.0", "method": "tx", "params": ["'"$base53"'", "contract.wormhole_crypto.near"]}' | sha256sum | head -c 64)
filename="$(dirname "$(realpath "$0")")/synthetic/$hash.json"
echo "$TX3" > "$filename"





# Generate data simulating an unfinalized block
# unfinalized/: same as synthetic/, but the block of height 76538232 (id: G3r7EszAnX2ecbV4jX8e7Ls9vamrwHnn19UP4SeUL5qv) ends up not getting finalized.
# This is simulated by making sure that 6zPnFkHojNQpbRgALHgRnbzhFvp55hido4Gv645nR8zf doesn't show up in any last_final_block and also

cp -r synthetic/ unfinalized
find "$(dirname "$(realpath "$0")")/unfinalized/" -type f -name '*.json' -print0 | xargs -r -0 sed -i -e 's/"last_final_block": "G3r7EszAnX2ecbV4jX8e7Ls9vamrwHnn19UP4SeUL5qv"/"last_final_block": "Ad7JSCXZTGegrfWLAmqupd1qiEEphpf5azfWayWCPS8G"/g'
find "$(dirname "$(realpath "$0")")/unfinalized/" -type f -name '*.json' -print0 | xargs -r -0 sed -i -e 's/"prev_hash": "G3r7EszAnX2ecbV4jX8e7Ls9vamrwHnn19UP4SeUL5qv"/"prev_hash": "Ad7JSCXZTGegrfWLAmqupd1qiEEphpf5azfWayWCPS8G"/g'
