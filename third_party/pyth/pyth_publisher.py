#!/usr/bin/env python3

from pyth_utils import *

import random
import sys
import threading
import time

# Accept connections from readiness probe
def publisher_readiness():
    run_or_die(["nc", "-k", "-l", "-p", READINESS_PORT])

# Update the specified price with random values
def publisher_random_update(price_pubkey):
    value = random.randrange(1024)
    confidence = 1
    pyth_run_or_die("upd_price_val", args=[price_pubkey, str(value), str(confidence), "trading"])
    print("Price updated!")

# Fund the publisher
sol_run_or_die("airdrop", [str(SOL_AIRDROP_AMT),
                           "--keypair", PYTH_PUBLISHER_KEYPAIR,
                           "--commitment", "finalized",
                           ])

# Create a mapping
pyth_run_or_die("init_mapping")

# Add a product
prod_pubkey = pyth_run_or_die("add_product", capture_output=True).stdout.strip()
print(f"Added product {prod_pubkey}")

# Add a price
price_pubkey = pyth_run_or_die(
    "add_price",
    args=[prod_pubkey, "price"],
    confirm=False,
    capture_output=True
).stdout.strip()

print(f"Added price {price_pubkey}")

publisher_pubkey = sol_run_or_die("address", args=["--keypair", PYTH_PUBLISHER_KEYPAIR], capture_output=True).stdout.strip()

# Become a publisher
pyth_run_or_die("add_publisher", args=[publisher_pubkey, price_pubkey], confirm=False, debug=True, capture_output=True)
print(f"Added publisher {publisher_pubkey}")

# Update the price as the newly added publisher
publisher_random_update(price_pubkey)

print(f"Updated price {price_pubkey}. Mock updates ready to roll. Updating every {str(PYTH_PUBLISHER_INTERVAL)} seconds")

# Spin off the readiness probe endpoint into a separate thread
readiness_thread = threading.Thread(target=publisher_readiness)

readiness_thread.start()

while True:
    print(f"Updating price {price_pubkey}")
    publisher_random_update(price_pubkey)
    time.sleep(PYTH_PUBLISHER_INTERVAL)
    sys.stdout.flush()

readiness_thread.join()
