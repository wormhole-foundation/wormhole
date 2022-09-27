// NOTE: This script only supports transferring tokens that originated on NEAR
// TODO: get_foreign_asset instead of hash_lookup for non-NEAR originating tokens :)

// Prerequisites
// cd ethereum && npm ci
// cd sdk/js && npm ci && npm run build
// cd near
// npm ci

// Run with
//   EVM_PK="" EVM_PROVIDER_URL="" EVM_CHAIN_NAME="" EVM_TOKEN="" TOKENS_TO_SEND="" NEAR_MNEMONIC="" NEAR_ACCOUNT="" npm run transferToNear
// or
//   EVM_PK="" EVM_PROVIDER_URL="" EVM_CHAIN_NAME="" EVM_TOKEN="" TOKENS_TO_SEND="" NEAR_PK="" NEAR_ACCOUNT="" npm run transferToNear

// for Eth try EVM_PROVIDER_URL="https://rpc.ankr.com/eth" and EVM_CHAIN_NAME="ethereum"
// for BSC try EVM_PROVIDER_URL="https://rpc.ankr.com/bsc" and EVM_CHAIN_NAME="bsc"

// It is SUPER SUPER important to use the near-api-js that comes from inside wormhole-sdk or all heck breaks lose
import {
  Account as nearAccount,
  connect as nearConnect,
  keyStores as nearKeyStores,
  providers as nearProviders,
  utils as nearUtils,
} from "@certusone/wormhole-sdk/node_modules/near-api-js";

import {
  approveEth,
  Bridge__factory,
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_NEAR,
  coalesceChainId,
  CONTRACTS,
  EVMChainName,
  getEmitterAddressEth,
  getOriginalAssetEth,
  getSignedVAAWithRetry,
  hexToUint8Array,
  isChain,
  isEVMChain,
  parseSequenceFromLogEth,
  redeemOnNear,
  transferFromEth,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import colors from "@colors/colors/safe";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import BN from "bn.js";
import { ethers } from "ethers";
// @ts-ignore
import { parseSeedPhrase } from "near-seed-phrase";
import prompt from "prompt";

export const WORMHOLE_RPC_HOSTS = [
  "https://wormhole-v2-mainnet-api.certus.one",
  "https://wormhole.inotel.ro",
  "https://wormhole-v2-mainnet-api.mcf.rocks",
  "https://wormhole-v2-mainnet-api.chainlayer.network",
  "https://wormhole-v2-mainnet-api.staking.fund",
  "https://wormhole-v2-mainnet.01node.com",
];

if (!process.env.EVM_PK) {
  console.log("EVM_PK is required");
  process.exit(1);
}
if (!process.env.EVM_PROVIDER_URL) {
  console.log(
    "EVM_PROVIDER_URL is required (try https://rpc.ankr.com/eth for Eth)"
  );
  process.exit(1);
}
if (
  !process.env.EVM_CHAIN_NAME ||
  !isChain(process.env.EVM_CHAIN_NAME) ||
  !isEVMChain(process.env.EVM_CHAIN_NAME) ||
  !CONTRACTS.MAINNET[process.env.EVM_CHAIN_NAME].core ||
  !CONTRACTS.MAINNET[process.env.EVM_CHAIN_NAME].token_bridge
) {
  console.log(
    "EVM_CHAIN_NAME is required and must be a valid Wormhole EVM chain name"
  );
  process.exit(1);
}
if (!process.env.EVM_TOKEN) {
  console.log("EVM_TOKEN is required");
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

const EVM_PK: string = process.env.EVM_PK;
const EVM_TOKEN: string = process.env.EVM_TOKEN;
const CHAIN_NAME: EVMChainName = process.env.EVM_CHAIN_NAME;
const CHAIN_ID: ChainId = coalesceChainId(process.env.EVM_CHAIN_NAME);
const TOKENS_TO_SEND: bigint = BigInt(process.env.TOKENS_TO_SEND);

async function transferTest() {
  let provider = new ethers.providers.JsonRpcProvider(
    process.env.EVM_PROVIDER_URL
  );
  let signer = new ethers.Wallet(EVM_PK, provider);
  let bridge = Bridge__factory.connect(
    CONTRACTS.MAINNET[CHAIN_NAME].token_bridge as string,
    signer
  );

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
    EVM_TOKEN,
    "from",
    await signer.getAddress(),
    "to",
    process.env.NEAR_ACCOUNT as string,
    "on Near"
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

  const { assetAddress, chainId } = await getOriginalAssetEth(
    CONTRACTS.MAINNET[CHAIN_NAME].token_bridge as string,
    provider,
    EVM_TOKEN,
    CHAIN_NAME
  );
  console.log("Original asset:", chainId, uint8ArrayToHex(assetAddress));

  const [success, nearTokenContract] = await userAccount.viewFunction(
    CONTRACTS.MAINNET.near.token_bridge,
    "hash_lookup",
    { hash: uint8ArrayToHex(assetAddress) }
  );

  console.log("Near token contract:", nearTokenContract);

  const initialBalance = await userAccount.viewFunction(
    nearTokenContract,
    "ft_balance_of",
    { account_id: process.env.NEAR_ACCOUNT as string }
  );

  console.log(
    "Balance of",
    nearTokenContract,
    "for",
    process.env.NEAR_ACCOUNT,
    ":",
    initialBalance
  );

  // So, near can have account names up to 64 bytes but wormhole can only have 32...
  //   as a result, we have to hash our account names to sha256's..  What we are doing
  //   here is doing a RPC call (does not require any interaction with the wallet and is free)
  //   that both tells us our account hash AND if we are already registered...
  let account_hash = await userAccount.viewFunction(
    CONTRACTS.MAINNET.near.token_bridge,
    "hash_account",
    {
      account: userAccount.accountId,
    }
  );

  console.log(account_hash);

  let myAddress = account_hash[1];

  if (!account_hash[0]) {
    console.log("Registering the receiving account");

    let myAddress2 = nearProviders.getTransactionLastResult(
      await userAccount.functionCall({
        contractId: CONTRACTS.MAINNET.near.token_bridge,
        methodName: "register_account",
        args: { account: process.env.NEAR_ACCOUNT as string },
        gas: new BN("100000000000000"),
        attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
      })
    );

    console.log("account hash returned: " + myAddress2);
  } else {
    console.log("account already registered");
  }

  console.log("Approving...");
  // approve the bridge to spend tokens
  await approveEth(
    CONTRACTS.MAINNET[CHAIN_NAME].token_bridge as string,
    EVM_TOKEN,
    signer,
    TOKENS_TO_SEND
  );
  console.log("Transferring...");
  // transfer tokens
  let receipt = await transferFromEth(
    CONTRACTS.MAINNET[CHAIN_NAME].token_bridge as string,
    signer,
    EVM_TOKEN,
    TOKENS_TO_SEND,
    CHAIN_ID_NEAR,
    hexToUint8Array(myAddress)
  );

  console.log("EVM tx submitted", receipt.transactionHash);

  const sequence = await parseSequenceFromLogEth(
    receipt,
    CONTRACTS.MAINNET[CHAIN_NAME].core as string
  );

  console.log(sequence);

  const emitterAddress = getEmitterAddressEth(
    CONTRACTS.MAINNET[CHAIN_NAME].token_bridge as string
  );

  console.log("emitterAddress:", emitterAddress, "sequence:", sequence);

  console.log(
    `If this script hangs, try https://wormhole-v2-mainnet-api.certus.one/v1/signed_vaa/${CHAIN_ID}/${emitterAddress}/${sequence.toString()}`
  );

  const confirmationsRequired = await bridge.finality();
  console.log(
    "Please be patient. Waiting",
    confirmationsRequired,
    "confirmations..."
  );

  const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    CHAIN_ID,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(),
    }
  );

  console.log("VAA received!");
  console.log(uint8ArrayToHex(signedVAA));

  const redeemMsgs = await redeemOnNear(
    userAccount.connection.provider,
    userAccount.accountId,
    CONTRACTS.MAINNET.near.token_bridge,
    signedVAA
  );
  for (const msg of redeemMsgs) {
    await userAccount.functionCall(msg);
  }

  console.log("Redeemed!");

  const endingBalance = await userAccount.viewFunction(
    nearTokenContract,
    "ft_balance_of",
    { account_id: process.env.NEAR_ACCOUNT as string }
  );

  console.log("Balance of", nearTokenContract, "for", process.env.NEAR_ACCOUNT);
  console.log("Before:", initialBalance);
  console.log("After:", endingBalance);
}

transferTest();
