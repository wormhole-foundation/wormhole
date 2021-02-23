#!/usr/bin/env bash
# Generate test lockups on Solana to be executed on Ethereum

# Constants (hardcoded)
eevaa_program_address=EevaaBridgeeXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o
recipient_address=90F8bf6A479f320ead074411a4B0e7944Ea8c9C1

eevaa=ABCDEF



while : ; do
  cli post-eevaa "$eevaa_program_address" "$eevaa" 
  sleep 5
done
