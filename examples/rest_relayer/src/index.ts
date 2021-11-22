import {
  CHAIN_ID_ETH,
  hexToUint8Array,
  redeemOnEth,
} from "@certusone/wormhole-sdk";
import { ethers } from "ethers";
require("dotenv").config();
if (!process.env.ETH_NODE_URL) {
  console.error("Missing environment variable ETH_NODE_URL");
  process.exit(1);
}
if (!process.env.ETH_PRIVATE_KEY) {
  console.error("Missing environment variable ETH_PRIVATE_KEY");
  process.exit(1);
}
if (!process.env.ETH_TOKEN_BRIDGE_ADDRESS) {
  console.error("Missing environment variable ETH_TOKEN_BRIDGE_ADDRESS");
  process.exit(1);
}
const SUPPORTED_CHAINS = [CHAIN_ID_ETH];
const express = require("express");
const app = express();
const bodyParser = require("body-parser");
app.use(bodyParser.urlencoded({ extended: true }));
app.use(bodyParser.json());
app.post("/relay", async (req, res) => {
  console.log(req.body);
  const chainId = req.body?.chainId;
  if (!SUPPORTED_CHAINS.includes(chainId)) {
    res.status(400).json({ error: "Unsupported chainId" });
    return;
  }
  const signedVAA = req.body?.signedVAA;
  if (!signedVAA) {
    res.status(400).json({ error: "signedVAA is required" });
  }
  const provider = new ethers.providers.WebSocketProvider(
    process.env.ETH_NODE_URL
  );
  const signer = new ethers.Wallet(process.env.ETH_PRIVATE_KEY, provider);
  const receipt = await redeemOnEth(
    process.env.ETH_TOKEN_BRIDGE_ADDRESS,
    signer,
    hexToUint8Array(signedVAA)
  );
  provider.destroy();
  res.status(200).json(receipt);
});
app.listen(3001, () => {
  console.log("Server running on port 3001");
});
