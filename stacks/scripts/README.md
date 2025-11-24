# scripts

## Deploy Wormhole Core in Stacks

Edit `.env`

```bash
cd stacks/scripts
npm install
bun run deployTestnet.ts
```
## Other scripts

* `postMessage.ts` - Sends a message via Wormhole core bridge
* `getGuardianSet.ts` - Prints the current guardian set
* `transferSTXNewAddress.ts` - Transfers STX to a new address and prints keys
