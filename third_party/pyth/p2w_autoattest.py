#!/usr/bin/env python3

# This script sets up a simple loop for periodical attestation of Pyth data
import json
import logging
import os
import re
import sys
import threading
import time
from http.client import HTTPConnection
from http.server import BaseHTTPRequestHandler, HTTPServer

from pyth_utils import *

logging.basicConfig(
    level=logging.DEBUG, format="%(asctime)s | %(module)s | %(levelname)s | %(message)s"
)

P2W_SOL_ADDRESS = os.environ.get(
    "P2W_SOL_ADDRESS", "P2WH424242424242424242424242424242424242424"
)
P2W_ATTEST_INTERVAL = float(os.environ.get("P2W_ATTEST_INTERVAL", 5))
P2W_OWNER_KEYPAIR = os.environ.get(
    "P2W_OWNER_KEYPAIR", "/usr/src/solana/keys/p2w_owner.json"
)
P2W_ATTESTATIONS_PORT = int(os.environ.get("P2W_ATTESTATIONS_PORT", 4343))
P2W_INITIALIZE_SOL_CONTRACT = os.environ.get("P2W_INITIALIZE_SOL_CONTRACT", None)

PYTH_TEST_ACCOUNTS_HOST = "pyth"
PYTH_TEST_ACCOUNTS_PORT = 4242

P2W_ATTESTATION_CFG = os.environ.get("P2W_ATTESTATION_CFG", None)

WORMHOLE_ADDRESS = os.environ.get(
    "WORMHOLE_ADDRESS", "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
)

ATTESTATIONS = {
    "pendingSeqnos": [],
}


class P2WAutoattestStatusEndpoint(BaseHTTPRequestHandler):
    """
    A dumb endpoint for last attested price metadata.
    """

    def do_GET(self):
        logging.info(f"Got path {self.path}")
        sys.stdout.flush()
        data = json.dumps(ATTESTATIONS).encode("utf-8")
        logging.debug(f"Sending: {data}")

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
    server_address = ("", P2W_ATTESTATIONS_PORT)
    httpd = HTTPServer(server_address, P2WAutoattestStatusEndpoint)
    httpd.serve_forever()

if SOL_AIRDROP_AMT > 0:
    # Fund the p2w owner
    sol_run_or_die("airdrop", [
        str(SOL_AIRDROP_AMT),
        "--keypair", P2W_OWNER_KEYPAIR,
        "--commitment", "finalized",
    ])
    

if P2W_INITIALIZE_SOL_CONTRACT is not None:
    # Get actor pubkeys
    P2W_OWNER_ADDRESS = sol_run_or_die(
        "address", ["--keypair", P2W_OWNER_KEYPAIR], capture_output=True
    ).stdout.strip()
    PYTH_OWNER_ADDRESS = sol_run_or_die(
        "address", ["--keypair", PYTH_PROGRAM_KEYPAIR], capture_output=True
    ).stdout.strip()

    init_result = run_or_die(
        [
            "pyth2wormhole-client",
            "--log-level",
            "4",
            "--p2w-addr",
            P2W_SOL_ADDRESS,
            "--rpc-url",
            SOL_RPC_URL,
            "--payer",
            P2W_OWNER_KEYPAIR,
            "init",
            "--wh-prog",
            WORMHOLE_ADDRESS,
            "--owner",
            P2W_OWNER_ADDRESS,
            "--pyth-owner",
            PYTH_OWNER_ADDRESS,
        ],
        capture_output=True,
        die=False,
    )

    if init_result.returncode != 0:
        logging.error(
            "NOTE: pyth2wormhole-client init failed, retrying with set_config"
        )
        run_or_die(
            [
                "pyth2wormhole-client",
                "--log-level",
                "4",
                "--p2w-addr",
                P2W_SOL_ADDRESS,
                "--rpc-url",
                SOL_RPC_URL,
                "--payer",
                P2W_OWNER_KEYPAIR,
                "set-config",
                "--owner",
                P2W_OWNER_KEYPAIR,
                "--new-owner",
                P2W_OWNER_ADDRESS,
                "--new-wh-prog",
                WORMHOLE_ADDRESS,
                "--new-pyth-owner",
                PYTH_OWNER_ADDRESS,
            ],
            capture_output=True,
        )

# Retrieve available symbols from the test pyth publisher if not provided in envs
if P2W_ATTESTATION_CFG is None:
    P2W_ATTESTATION_CFG = "./attestation_cfg_test.yaml"
    conn = HTTPConnection(PYTH_TEST_ACCOUNTS_HOST, PYTH_TEST_ACCOUNTS_PORT)

    conn.request("GET", "/")

    res = conn.getresponse()

    pyth_accounts = None

    if res.getheader("Content-Type") == "application/json":
        pyth_accounts = json.load(res)
    else:
        logging.error("Bad Content type")
        sys.exit(1)

    cfg_yaml = f"""
---
symbols:"""

    logging.info(f"Retrieved {len(pyth_accounts)} Pyth accounts from endpoint: {pyth_accounts}")

    for acc in pyth_accounts:

        name = acc["name"]
        price = acc["price"]
        product = acc["product"]

        cfg_yaml += f"""
    - name: {name}
      price_addr: {price}
      product_addr: {product}"""

    with open(P2W_ATTESTATION_CFG, "w") as f:
        f.write(cfg_yaml)
        f.flush()
        

attest_result = run_or_die(
    [
        "pyth2wormhole-client",
        "--log-level",
        "4",
        "--p2w-addr",
        P2W_SOL_ADDRESS,
        "--rpc-url",
        SOL_RPC_URL,
        "--payer",
        P2W_OWNER_KEYPAIR,
        "attest",
        "-f",
        P2W_ATTESTATION_CFG
        
    ],
    capture_output=True,
)

logging.info("p2w_autoattest ready to roll!")
logging.info(f"Attest Interval: {P2W_ATTEST_INTERVAL}")

# Serve p2w endpoint
endpoint_thread = threading.Thread(target=serve_attestations, daemon=True)
endpoint_thread.start()

# Let k8s know the service is up
readiness_thread = threading.Thread(target=readiness, daemon=True)
readiness_thread.start()

seqno_regex = re.compile(r"Sequence number: (\d+)")

while True:
    matches = seqno_regex.findall(attest_result.stdout)

    seqnos = list(map(lambda m: int(m), matches))

    ATTESTATIONS["pendingSeqnos"] += seqnos

    logging.info(f"{len(seqnos)} batch seqno(s) received: {seqnos})")

    attest_result = run_or_die(
        [
            "pyth2wormhole-client",
            "--log-level",
            "4",
            "--p2w-addr",
            P2W_SOL_ADDRESS,
            "--rpc-url",
            SOL_RPC_URL,
            "--payer",
            P2W_OWNER_KEYPAIR,
            "attest",
            "-f",
            P2W_ATTESTATION_CFG
        ],
        capture_output=True,
    )
    time.sleep(P2W_ATTEST_INTERVAL)
