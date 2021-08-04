#!/usr/bin/env python3

from pyth_utils import *

from http.server import HTTPServer, BaseHTTPRequestHandler

import json
import random
import sys
import threading
import time


class P2WAccEndpoint(BaseHTTPRequestHandler):
    """
    A dumb endpoint to respond with a JSON containing Pyth account addresses
    """

    def do_GET(self):
        print(f"Got path {self.path}")
        sys.stdout.flush()
        data = json.dumps(ACCOUNTS).encode("utf-8")
        print(f"Sending:\n{data}")

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
        self.wfile.flush()


ACCOUNTS = dict()


def publisher_random_update(price_pubkey):
    """
    Update the specified price with random values
    """
    value = random.randrange(1024)
    confidence = 1
    pyth_run_or_die("upd_price_val", args=[
        price_pubkey, str(value), str(confidence), "trading"
    ])
    print(f"Price {price_pubkey} value updated to {str(value)}!")


def accounts_endpoint():
    """
    Run a barebones HTTP server to share the dynamic Pyth
    mapping/product/price account addresses
    """
    server_address = ('', 4242)
    httpd = HTTPServer(server_address, P2WAccEndpoint)
    httpd.serve_forever()


# Fund the publisher
sol_run_or_die("airdrop", [
    str(SOL_AIRDROP_AMT),
    "--keypair", PYTH_PUBLISHER_KEYPAIR,
    "--commitment", "finalized",
])

# Create a mapping
pyth_run_or_die("init_mapping")

# Add a product
prod_pubkey = pyth_run_or_die(
    "add_product", capture_output=True).stdout.strip()
print(f"Added product {prod_pubkey}")

# Add a price
price_pubkey = pyth_run_or_die(
    "add_price",
    args=[prod_pubkey, "price"],
    confirm=False,
    capture_output=True
).stdout.strip()

print(f"Added price {price_pubkey}")

publisher_pubkey = sol_run_or_die("address", args=[
    "--keypair", PYTH_PUBLISHER_KEYPAIR
], capture_output=True).stdout.strip()

# Become a publisher
pyth_run_or_die(
    "add_publisher", args=[publisher_pubkey, price_pubkey],
    confirm=False,
    debug=True,
    capture_output=True)
print(f"Added publisher {publisher_pubkey}")

# Update the price as the newly added publisher
publisher_random_update(price_pubkey)

print(
    f"Mock updates ready to roll. Updating every {str(PYTH_PUBLISHER_INTERVAL)} seconds")

# Spin off the readiness probe endpoint into a separate thread
readiness_thread = threading.Thread(target=readiness, daemon=True)

# Start an HTTP endpoint for looking up product/price address
http_service = threading.Thread(target=accounts_endpoint, daemon=True)

ACCOUNTS["product"] = prod_pubkey
ACCOUNTS["price"] = price_pubkey

readiness_thread.start()
http_service.start()

while True:
    publisher_random_update(price_pubkey)
    time.sleep(PYTH_PUBLISHER_INTERVAL)
    sys.stdout.flush()

readiness_thread.join()
http_service.join()
