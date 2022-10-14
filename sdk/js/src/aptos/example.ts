import { AptosAccount } from "aptos";
import { WormholeAptosApi } from ".";
import { hex } from "../utils";
import { AptosClientWrapper } from "./client";

// it's ok if this key is published
const devnet_key = "537c1f91e56891445b491068f519b705f8c0f1a1e66111816dd5d4aa85b8113d";
const rpc = "http://0.0.0.0:8080";
const sender = new AptosAccount(new Uint8Array(Buffer.from(devnet_key, "hex")));
const client = new AptosClientWrapper(rpc);
const api = new WormholeAptosApi(client, "DEVNET");

// requires local validator to be running
// we expect this to fail if run twice on the same local validator instance
api.tokenBridge
  .registerChain(
    sender,
    hex(
      "0100000000010015d405c74be6d93c3c33ed6b48d8db70dfb31e0981f8098b2a6c7583083e0c3343d4a1abeb3fc1559674fa067b0c0e2e9de2fafeaecdfeae132de2c33c9d27cc0100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000016911ae00000000000000000000000000000000000000000000546f6b656e427269646765010000000200000000000000000000000000000000000000000000000000000000deadbeef",
    ),
  )
  .then(console.log);
