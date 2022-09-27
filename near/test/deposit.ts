const sha256 = require("js-sha256");
const fs = require("fs").promises;
const assert = require("assert").strict;
const fetch = require("node-fetch");
const elliptic = require("elliptic");
const web3Utils = require("web3-utils");
const BN = require("bn.js");
import { formatUnits, parseUnits } from "@ethersproject/units";

import { zeroPad } from "@ethersproject/bytes";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

const { parseSeedPhrase, generateSeedPhrase } = require("near-seed-phrase");

import {
  CHAIN_ID_NEAR,
  CHAIN_ID_ETH,
  hexToUint8Array,
} from "@certusone/wormhole-sdk/lib/cjs/utils";

export const WORMHOLE_RPC_HOSTS = [
  "https://wormhole-v2-mainnet-api.certus.one",
  "https://wormhole.inotel.ro",
  "https://wormhole-v2-mainnet-api.mcf.rocks",
  "https://wormhole-v2-mainnet-api.chainlayer.network",
  "https://wormhole-v2-mainnet-api.staking.fund",
  "https://wormhole-v2-mainnet.01node.com",
];

// It is SUPER SUPER important to use the near-api-js that comes from inside wormhole-sdk or all heck breaks lose
import {
  connect as nearConnect,
  keyStores as nearKeyStores,
  utils as nearUtils,
  Account as nearAccount,
  providers as nearProviders,
} from "@certusone/wormhole-sdk/node_modules/near-api-js";

import {
  CONTRACTS,
  attestNearFromNear,
  attestTokenFromNear,
  createWrappedOnNear,
  getForeignAssetNear,
  getIsTransferCompletedNear,
  getIsWrappedAssetNear,
  getOriginalAssetNear,
  getSignedVAAWithRetry,
  redeemOnNear,
  transferFromEth,
  transferNearFromNear,
  transferTokenFromNear,
  getEmitterAddressEth,
  parseSequenceFromLogEth,
} from "@certusone/wormhole-sdk";

const sdk = require("@certusone/wormhole-sdk");
const { ethers } = require("ethers");

async function transferTest() {
  let provider = new ethers.providers.JsonRpcProvider(
    "https://rpc.ankr.com/eth"
  );
  let signer = new ethers.Wallet(process.env.ETH_PK, provider);
  let bridge = sdk.Bridge__factory.connect(
    "0x3ee18B2214AFF97000D974cf647E7C347E8fa585",
    signer
  );

  let nearNodeUrl = "https://rpc.mainnet.near.org";
  let portalAccount = "contract.portalbridge.near";
  let networkId = "mainnet";

  // There are many kinds of keystores...  in this case, I am using a InMemory one
  let userKeys = parseSeedPhrase(process.env.NEAR_KEYS);
  let userKey = nearUtils.KeyPair.fromString(userKeys["secretKey"]);
  let keyStore = new nearKeyStores.InMemoryKeyStore();

  keyStore.setKey(networkId, process.env.NEAR_ACCOUNT as string, userKey);

  // connect to near...
  let near = await nearConnect({
    headers: {},
    keyStore,
    networkId: networkId as string,
    nodeUrl: nearNodeUrl as string,
  });

  console.log(process.env.NEAR_ACCOUNT as string);
  // rpc handle
  const userAccount = new nearAccount(
    near.connection,
    process.env.NEAR_ACCOUNT as string
  );

  // So, near can have account names up to 64 bytes but wormhole can only have 32...
  //   as a result, we have to hash our account names to sha256's..  What we are doing
  //   here is doing a RPC call (does not require any interaction with the wallet and is free)
  //   that both tells us our account hash AND if we are already registered...
  let account_hash = await userAccount.viewFunction(
    portalAccount,
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
        contractId: portalAccount,
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

  // like this
  await redeemOnNear(userAccount, token_bridge, hexToUint8Array(trans));
}

transferTest();
