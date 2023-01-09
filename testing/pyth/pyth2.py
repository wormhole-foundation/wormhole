import argparse
import csv
import json
import base64
import struct
import time
import sys

import borsh
from borsh import types
import websocket  # pip install websocket-client

# 	MessagePublicationAccount struct {
# 		VaaVersion uint8
# 		// Borsh does not seem to support booleans, so 0=false / 1=true
# 		ConsistencyLevel    uint8
# 		VaaTime             uint32
# 		VaaSignatureAccount vaa.Address
# 		SubmissionTime      uint32
# 		Nonce               uint32
# 		Sequence            uint64
# 		EmitterChain        uint16
# 		EmitterAddress      vaa.Address
# 		Payload             []byte
# 	}

wormhole_schema = borsh.schema(
    {
        "VaaVersion": types.u8,
        "ConsistencyLevel": types.u8,
        "VaaTime": types.u32,
        "VaaSignatureAccount": types.fixed_array(types.u8, 32),
        "SubmissionTime": types.u32,
        "Nonce": types.u32,
        "Sequence": types.u64,
        "EmitterChain": types.u16,
        "EmitterAddress": types.fixed_array(types.u8, 32),
        # "Payload": types.dynamic_array(types.u8),
    }
)


class ServerException(Exception):
    """Raised when the JSON-RPC server returns an error."""

    def __init__(self, code, message):
        self.code = code
        self.message = message
        super().__init__(self.message)


class PythnetVAAPubSubClient:
    def __init__(self, endpoint, program_id, commitment):
        self.ws = websocket.WebSocketApp(
            endpoint,
            on_message=lambda _, message: self.on_message(message),
            on_open=lambda _: self.on_open(),
            on_error=lambda _, error: self.on_error(error),
            on_close=lambda _, close_status_code, close_msg: self.on_close(
                close_status_code, close_msg
            ),
        )
        self.program_id = program_id
        self.commitment = commitment

    def on_message(self, message):
        msg = json.loads(message)
        print(json.dumps(msg, indent=2))

    def on_open(self):
        print("[~] Connection Established", file=sys.stderr)

        self.ws.send(
            json.dumps(
                {
                    "jsonrpc": "2.0",
                    "id": 1,
                    "method": "blockSubscribe",
                    "params": [
                        {
                            "mentionsAccountOrProgram":  self.program_id,
                        },
                        {
                            "encoding": "base64",
                            "commitment": "confirmed",
                            "showRewards": True,
                            "transactionDetails": "full"
                        }
                    ],
                }
            )
        )

    def on_error(self, error):
        """Callback by WebSocket client when an exception occurs."""
        if isinstance(error, websocket.WebSocketException):
            # Treat WebSocket errors as ephemeral and retry
            print("[!]", error, file=sys.stderr)
        else:
            # Rethrow to abort
            raise error

    def on_close(self, close_status_code, close_msg):
        """Callback by WebSocket client when a connection closes."""
        print("[!] Connection Closed", file=sys.stderr)

    def run(self):
        """Runs the WebSocket client forever."""
        try:
            self.ws.run_forever(ping_interval=5)
        except KeyboardInterrupt:
            print("[~] Shutting down")
            pass


def _main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--pubsub_url",
        type=str,
        default="wss://pythnet.rpcpool.com",
        help="WebSocket URL to Pythnet PubSub API",
    )
    parser.add_argument(
        "--pyth_program",
        type=str,
        default="H3fxXJ86ADW2PNuDDmZJg6mzTtPxkYCpNuQUTgmJ7AjU",
        help="Program ID of Pyth VAA aggregator",
    )
    parser.add_argument(
        "--commitment",
        type=str,
        default=["confirmed"],
        choices=["processed", "confirmed", "finalized"],
        help="Solana commitment level",
    )
    args = parser.parse_args()

    client = PythnetVAAPubSubClient(args.pubsub_url, args.pyth_program, args.commitment)
    client.run()


if __name__ == "__main__":
    _main()
