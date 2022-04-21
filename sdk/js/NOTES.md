- install python3 (should already be installed)
- install pip (sudo apt-get install pip)
- cd ~/git/algoIntegration/wormhole/algorand
- sudo python3 -m pip install -r ~/git/algoIntegration/wormhole/algorand/requirements.txt
- The following line will reset the sandbox and create the correct coreId and tokenBridgeId
- ./sandbox reset -v; python3 admin.py --devnet --boot

If the sandbox isn't up yet: ./sandbox up -v dev

Proposal for filesystem:

add:  sdk/js/src/algorand
add:  sdk/js/src/algorand/__tests__
