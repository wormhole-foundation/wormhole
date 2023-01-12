import websocket
import _thread
import time
import rel
import os
import pprint
import json
import base64

# https://github.com/MystenLabs/sui/pull/5113

# {
#     "jsonrpc": "2.0",
#     "method": "sui_subscribeEvent",
#     "params": {
#         "subscription": 1805876586195140,
#         "result": {
#             "timestamp": 1666704112752,
#             "txDigest": "ckB13AaG+OHrO0Ha3I8IK3ERanYHmHAI0jSXnqk9R+I=",
#             "event": {
#                 "moveEvent": {
#                     "packageId": "0xbd99019f3c8f9d08b5498fedcc97e1c24cddff88",
#                     "transactionModule": "wormhole",
#                     "sender": "0xdc2f7334400a353c6a9303235b578477202809c6",
#                     "type": "0xbd99019f3c8f9d08b5498fedcc97e1c24cddff88::state::WormholeMessage",
#                     "fields": {
#                         "consistency_level": 0,
#                         "nonce": 400,
#                         "payload": "Ag==",
#                         "sender": "0xdc2f7334400a353c6a9303235b578477202809c6",
#                         "sequence": 19,
#                         "timestamp": 0
#                     },
#                     "bcs": "3C9zNEAKNTxqkwMjW1eEdyAoCcYTAAAAAAAAAJABAAAAAAAAAQIAAAAAAAAAAAA="
#                 }
#             }
#         }
#     }
# }

# curl -s -X POST -d '{"jsonrpc":"2.0", "id": 1, "method": "sui_getEventsByTransaction", "params": ["KgsiF8pCF61N02zX2oMFYLWQdrbxkOD1ypBxND752No=", 2]}' -H 'Content-Type: application/json' http://127.0.0.1:9000 | jq

# {
#   "jsonrpc": "2.0",
#   "result": [
#     {
#       "timestamp": 1666704112752,
#       "txDigest": "ckB13AaG+OHrO0Ha3I8IK3ERanYHmHAI0jSXnqk9R+I=",
#       "event": {
#         "moveEvent": {
#           "packageId": "0xbd99019f3c8f9d08b5498fedcc97e1c24cddff88",
#           "transactionModule": "wormhole",
#           "sender": "0xdc2f7334400a353c6a9303235b578477202809c6",
#           "type": "0xbd99019f3c8f9d08b5498fedcc97e1c24cddff88::state::WormholeMessage",
#           "bcs": "3C9zNEAKNTxqkwMjW1eEdyAoCcYTAAAAAAAAAJABAAAAAAAAAQIAAAAAAAAAAAA="
#         }
#       }
#     }
#   ],
#   "id": 1
# }

def on_message(ws, message):
    v = json.loads(message)
    print(json.dumps(v, indent=4))
    if "params" in v:
        tx = v["params"]["result"]["txDigest"]
        #tx = base64.standard_b64decode(tx)
        print(tx + " -> " + base64.standard_b64decode(tx).hex())

        pl = v["params"]["result"]["event"]["moveEvent"]["fields"]["payload"]
        pl = base64.standard_b64decode(pl)
        print(pl.hex())

def on_error(ws, error):
    print(error)

def on_close(ws, close_status_code, close_msg):
    print("### closed ###")

def on_open(ws):
    print("Opened connection")
    ws.send("{\"jsonrpc\":\"2.0\", \"id\": 1, \"method\": \"sui_subscribeEvent\", \"params\": [{\"Package\": \"" + os.getenv("WORM_PACKAGE") + "\"}]}")

if __name__ == "__main__":
    ws = websocket.WebSocketApp("ws://localhost:9001",
                              on_open=on_open,
                              on_message=on_message,
                              on_error=on_error,
                              on_close=on_close)

    ws.run_forever(dispatcher=rel)  # Set dispatcher to automatic reconnection
    rel.signal(2, rel.abort)  # Keyboard Interrupt
    rel.dispatch()
