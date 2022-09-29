// Prerequisites
// cd ethereum && npm ci
// cd sdk/js && npm ci && npm run build
// cd near
// npm ci

// Run with
//   ETH_ADDRESS="" NEAR_TOKEN="" TOKENS_TO_SEND="" NEAR_MNEMONIC="" NEAR_ACCOUNT="" npm run transferFromNear
// or
//   ETH_ADDRESS="" NEAR_TOKEN="" TOKENS_TO_SEND="" NEAR_PK="" NEAR_ACCOUNT="" npm run transferFromNear

// It is SUPER SUPER important to use the near-api-js that comes from inside wormhole-sdk or all heck breaks lose
import {
  Account as nearAccount,
  connect as nearConnect,
  keyStores as nearKeyStores,
  utils as nearUtils,
} from "@certusone/wormhole-sdk/node_modules/near-api-js";

import {
  CHAIN_ID_ETH,
  CHAIN_ID_NEAR,
  CONTRACTS,
  getEmitterAddressEth,
  getEmitterAddressNear,
  getSignedVAAWithRetry,
  hexToUint8Array,
  parseSequenceFromLogNear,
  transferTokenFromNear,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
// @ts-ignore
import { parseSeedPhrase } from "near-seed-phrase";
import colors from "@colors/colors/safe";
import prompt from "prompt";

export const WORMHOLE_RPC_HOSTS = [
  "https://wormhole-v2-mainnet-api.certus.one",
  "https://wormhole.inotel.ro",
  "https://wormhole-v2-mainnet-api.mcf.rocks",
  "https://wormhole-v2-mainnet-api.chainlayer.network",
  "https://wormhole-v2-mainnet-api.staking.fund",
  "https://wormhole-v2-mainnet.01node.com",
];

if (!process.env.ETH_ADDRESS) {
  console.log("ETH_ADDRESS is required");
  process.exit(1);
}
if (!process.env.NEAR_TOKEN) {
  console.log("NEAR_TOKEN is required");
  process.exit(1);
}
if (!process.env.NEAR_ACCOUNT) {
  console.log("NEAR_ACCOUNT is required");
  process.exit(1);
}
if (!process.env.TOKENS_TO_SEND) {
  console.log("TOKENS_TO_SEND is required");
  process.exit(1);
}

const YOUR_ETH_ADDRESS_HERE = getEmitterAddressEth(process.env.ETH_ADDRESS);
const NEAR_TOKEN_ADDRESS: string = process.env.NEAR_TOKEN;
const TOKENS_TO_SEND: bigint = BigInt(process.env.TOKENS_TO_SEND);

async function transferTest() {
  let nearNodeUrl = "https://rpc.mainnet.near.org";
  let networkId = "mainnet";

  // There are many kinds of keystores...  in this case, I am using a InMemory one
  let keyStore = new nearKeyStores.InMemoryKeyStore();

  if (process.env.NEAR_MNEMONIC) {
    let userKeys = parseSeedPhrase(process.env.NEAR_MNEMONIC);
    let userKey = nearUtils.KeyPair.fromString(userKeys["secretKey"]);
    keyStore.setKey(networkId, process.env.NEAR_ACCOUNT as string, userKey);
  } else if (process.env.NEAR_PK) {
    let userKey = nearUtils.KeyPair.fromString(process.env.NEAR_PK);
    keyStore.setKey(networkId, process.env.NEAR_ACCOUNT as string, userKey);
  } else {
    console.log("NEAR_MNEMONIC or NEAR_PK is required");
    process.exit(1);
  }

  // connect to near...
  let near = await nearConnect({
    headers: {},
    keyStore,
    networkId: networkId as string,
    nodeUrl: nearNodeUrl as string,
  });

  console.log(
    "Sending",
    TOKENS_TO_SEND.toString(),
    NEAR_TOKEN_ADDRESS,
    "from",
    process.env.NEAR_ACCOUNT as string,
    "to",
    YOUR_ETH_ADDRESS_HERE,
    "on Ethereum"
  );

  prompt.message = "";
  const { input } = await prompt.get({
    properties: {
      input: {
        description: colors.red(
          "Are you sure you want to send tokens? THIS CANNOT BE UNDONE! [y/N]"
        ),
      },
    },
  });
  if (input !== "y") return;

  // rpc handle
  const userAccount = new nearAccount(
    near.connection,
    process.env.NEAR_ACCOUNT as string
  );
  const provider = userAccount.connection.provider;

  const transferMsgs = await transferTokenFromNear(
    provider,
    userAccount.accountId,
    CONTRACTS.MAINNET.near.core,
    CONTRACTS.MAINNET.near.token_bridge,
    NEAR_TOKEN_ADDRESS,
    TOKENS_TO_SEND,
    hexToUint8Array(YOUR_ETH_ADDRESS_HERE),
    CHAIN_ID_ETH,
    BigInt(0)
  );
  let transferOutcome;
  for (const msg of transferMsgs) {
    transferOutcome = await userAccount.functionCall(msg);
  }
  const sequence = parseSequenceFromLogNear(transferOutcome);
  if (sequence === null) {
    console.log("No sequence found, check above for error");
    process.exit(1);
  }

  const emitterAddress = getEmitterAddressNear(
    CONTRACTS.MAINNET.near.token_bridge
  );
  console.log("emitterAddress:", emitterAddress, "sequence:", sequence);

  console.log(
    `If this script hangs, try https://wormhole-v2-mainnet-api.certus.one/v1/signed_vaa/15/${emitterAddress}/${sequence.toString()}`
  );

  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID_NEAR,
    emitterAddress,
    sequence.toString(),
    {
      transport: NodeHttpTransport(),
    }
  );

  console.log("Redeem this on https://www.portalbridge.com/#/redeem");
  console.log(`Expand "Advanced" and paste this Signed VAA`);
  console.log(uint8ArrayToHex(signedVAA));
}

transferTest();
