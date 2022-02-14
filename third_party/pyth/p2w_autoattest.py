#!/usr/bin/env python3

# This script sets up a simple loop for periodical attestation of Pyth data
from pyth_utils import *

from http.client import HTTPConnection
from http.server import HTTPServer, BaseHTTPRequestHandler

import json
import os
import re
import subprocess
import time
import threading


P2W_ADDRESS = "P2WH424242424242424242424242424242424242424"
P2W_ATTEST_INTERVAL = float(os.environ.get("P2W_ATTEST_INTERVAL", 5))
P2W_OWNER_KEYPAIR = os.environ.get(
    "P2W_OWNER_KEYPAIR", f"/usr/src/solana/keys/p2w_owner.json")
P2W_ATTESTATIONS_PORT = int(os.environ.get("P2W_ATTESTATIONS_PORT", 4343))

PYTH_ACCOUNTS_HOST = "pyth"
PYTH_ACCOUNTS_PORT = 4242

WORMHOLE_ADDRESS = "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"

ATTESTATIONS = {
    "pendingSeqnos": [],
}

class P2WAutoattestStatusEndpoint(BaseHTTPRequestHandler):
    """
    A dumb endpoint for last attested price metadata.
    """

    def do_GET(self):
        print(f"Got path {self.path}")
        sys.stdout.flush()
        data = json.dumps(ATTESTATIONS).encode("utf-8")
        print(f"Sending:\n{data}")

        ATTESTATIONS["pendingSeqnos"] = []

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
        self.wfile.flush()

def serve_attestations():
    """
    Run a barebones HTTP server to share Pyth2wormhole attestation history
    """
    server_address = ('', P2W_ATTESTATIONS_PORT)
    httpd = HTTPServer(server_address, P2WAutoattestStatusEndpoint)
    httpd.serve_forever()


# Get actor pubkeys
P2W_OWNER_ADDRESS = sol_run_or_die(
    "address", ["--keypair", P2W_OWNER_KEYPAIR], capture_output=True).stdout.strip()
PYTH_OWNER_ADDRESS = sol_run_or_die(
    "address", ["--keypair", PYTH_PROGRAM_KEYPAIR], capture_output=True).stdout.strip()


# Top up pyth2wormhole owner
sol_run_or_die("airdrop", [
    str(SOL_AIRDROP_AMT),
    "--keypair",  P2W_OWNER_KEYPAIR,
    "--commitment", "finalized",
], capture_output=True)

# Initialize pyth2wormhole
init_result = run_or_die([
    "pyth2wormhole-client",
    "--log-level", "4",
    "--p2w-addr", P2W_ADDRESS,
    "--rpc-url", SOL_RPC_URL,
    "--payer", P2W_OWNER_KEYPAIR,
    "init",
    "--wh-prog", WORMHOLE_ADDRESS,
    "--owner", P2W_OWNER_ADDRESS,
    "--pyth-owner", PYTH_OWNER_ADDRESS,
], capture_output=True, die=False)

if init_result.returncode != 0:
    print("NOTE: pyth2wormhole-client init failed, retrying with set_config")
    run_or_die([
        "pyth2wormhole-client",
        "--log-level", "4",
        "--p2w-addr", P2W_ADDRESS,
        "--rpc-url", SOL_RPC_URL,
        "--payer", P2W_OWNER_KEYPAIR,
        "set-config",
        "--owner", P2W_OWNER_KEYPAIR,
        "--new-owner", P2W_OWNER_ADDRESS,
        "--new-wh-prog", WORMHOLE_ADDRESS,
        "--new-pyth-owner", PYTH_OWNER_ADDRESS,
    ], capture_output=True)

# Retrieve current price/product pubkeys from the pyth publisher
conn = HTTPConnection(PYTH_ACCOUNTS_HOST, PYTH_ACCOUNTS_PORT)

conn.request("GET", "/")

res = conn.getresponse()

pyth_accounts = None

if res.getheader("Content-Type") == "application/json":
    pyth_accounts = json.load(res)
else:
    print(f"Bad Content type {res.getheader('Content-Type')}", file=sys.stderr)
    sys.exit(1)

price_addr = pyth_accounts["price"]
product_addr = pyth_accounts["product"]

nonce = 0
attest_result = run_or_die([
    "pyth2wormhole-client",
    "--log-level", "4",
    "--p2w-addr", P2W_ADDRESS,
    "--rpc-url", SOL_RPC_URL,
    "--payer", P2W_OWNER_KEYPAIR,
    "attest",
    "--price", price_addr,
    "--product", product_addr,
    "--nonce", str(nonce),
], capture_output=True)

print("p2w_autoattest ready to roll.")
print(f"ACCOUNTS: {pyth_accounts}")
print(f"Attest Interval: {P2W_ATTEST_INTERVAL}")

# Serve p2w endpoint
endpoint_thread = threading.Thread(target=serve_attestations, daemon=True)
endpoint_thread.start()

# Let k8s know the service is up
readiness_thread = threading.Thread(target=readiness, daemon=True)
readiness_thread.start()

seqno_regex = re.compile(r"^Sequence number: (\d+)")

nonce = 1
while True:
    attest_result = run_or_die([
        "pyth2wormhole-client",
        "--log-level", "4",
        "--p2w-addr", P2W_ADDRESS,
        "--rpc-url", SOL_RPC_URL,
        "--payer", P2W_OWNER_KEYPAIR,
        "attest",
        "--price", price_addr,
        "--product", product_addr,
        "--nonce", str(nonce),
    ], capture_output=True)
    matches = seqno_regex.match(attest_result.stdout)

    if matches is not None:
        seqno = int(matches.group(1))
        print(f"Got seqno {seqno}")

        ATTESTATIONS["pendingSeqnos"].append(seqno)

    else:
        print(f"Warning: Could not get sequence number")

    time.sleep(P2W_ATTEST_INTERVAL)
    nonce += 1

readiness_thread.join()
