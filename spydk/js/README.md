# Wormhole Spy SDK

> Note: This is a pre-alpha release and in active development. Function names and signatures are subject to change.

Wormhole Spy service SDK for use with [@certusone/wormhole-sdk](https://www.npmjs.com/package/@certusone/wormhole-sdk)

## Usage

```js
import {
  createSpyRPCServiceClient,
  subscribeSignedVAA,
} from "@certusone/wormhole-spydk";
const client = createSpyRPCServiceClient(SPY_SERVICE_HOST);
const stream = await subscribeSignedVAA(client, {});
stream.on("data", ({ vaaBytes }) => {
  console.log(vaaBytes);
});
```

Also see [integration tests](https://github.com/wormhole-foundation/wormhole/blob/main/spydk/js/src/__tests__/integration.ts)
