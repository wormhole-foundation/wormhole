// npx pretty-quick

const nearAPI = require("near-api-js");
const fetch = require("node-fetch");
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

const BN = require("bn.js");

import { TestLib } from "./testlib";

import algosdk, {
  Account,
  decodeAddress,
} from "@certusone/wormhole-sdk/node_modules/algosdk";

import {
  getAlgoClient,
  getTempAccounts,
  signSendAndConfirmAlgorand,
} from "./algoHelpers";

import {
  CHAIN_ID_ALGORAND,
  CHAIN_ID_NEAR,
  ChainId,
} from "@certusone/wormhole-sdk/lib/cjs/utils";

import {
  CONTRACTS,
  createWrappedOnAlgorand,
  createWrappedOnNear,
  getEmitterAddressAlgorand,
  getForeignAssetAlgorand,
  getSignedVAAWithRetry,
  redeemOnAlgorand,
  redeemOnNear,
  transferFromAlgorand,
  transferTokenFromNear,
} from "@certusone/wormhole-sdk";

import { parseSequenceFromLogAlgorand } from "@certusone/wormhole-sdk/lib/cjs/bridge";

import { _parseVAAAlgorand } from "@certusone/wormhole-sdk/lib/cjs/algorand";
import {
  getEmitterAddressNear,
  parseSequenceFromLogNear,
} from "@certusone/wormhole-sdk/src";

export const uint8ArrayToHex = (a: Uint8Array): string =>
  Buffer.from(a).toString("hex");

export const hexToUint8Array = (h: string): Uint8Array =>
  new Uint8Array(Buffer.from(h, "hex"));

function getConfig(env: any) {
  switch (env) {
    case "sandbox":
    case "local":
      return {
        networkId: "sandbox",
        nodeUrl: "http://localhost:3030",
        masterAccount: "test.near",
        wormholeAccount: "wormhole.test.near",
        tokenAccount: "token.test.near",
        userAccount:
          Math.floor(Math.random() * 10000).toString() + "user.test.near",
        user2Account:
          Math.floor(Math.random() * 10000).toString() + "user.test.near",
      };
  }
  return {};
}

export function logNearGas(result: any, comment: string) {
  const { totalGasBurned, totalTokensBurned } = result.receipts_outcome.reduce(
    (acc: any, receipt: any) => {
      acc.totalGasBurned += receipt.outcome.gas_burnt;
      acc.totalTokensBurned += nearAPI.utils.format.formatNearAmount(
        receipt.outcome.tokens_burnt
      );
      return acc;
    },
    {
      totalGasBurned: result.transaction_outcome.outcome.gas_burnt,
      totalTokensBurned: nearAPI.utils.format.formatNearAmount(
        result.transaction_outcome.outcome.tokens_burnt
      ),
    }
  );
  console.log(
    comment,
    "totalGasBurned",
    totalGasBurned,
    "totalTokensBurned",
    totalTokensBurned
  );
}

async function testNearSDK() {
  let config = getConfig(process.env.NEAR_ENV || "sandbox");

  // Retrieve the validator key directly in the Tilt environment
  const response = await fetch("http://localhost:3031/validator_key.json");

  const keyFile = await response.json();

  let masterKey = nearAPI.utils.KeyPair.fromString(
    keyFile.secret_key || keyFile.private_key
  );

  let keyStore = new nearAPI.keyStores.InMemoryKeyStore();
  keyStore.setKey(config.networkId, config.masterAccount, masterKey);

  let near = await nearAPI.connect({
    deps: {
      keyStore,
    },
    networkId: config.networkId,
    nodeUrl: config.nodeUrl,
  });
  let masterAccount = new nearAPI.Account(
    near.connection,
    config.masterAccount
  );

  console.log(
    "Finish init NEAR masterAccount: " +
      JSON.stringify(await masterAccount.getAccountBalance())
  );

  let userKey = nearAPI.utils.KeyPair.fromRandom("ed25519");
  keyStore.setKey(config.networkId, config.userAccount, userKey);
  let user2Key = nearAPI.utils.KeyPair.fromRandom("ed25519");
  keyStore.setKey(config.networkId, config.user2Account, user2Key);

  console.log(
    "creating a user account: " +
      config.userAccount +
      " with key " +
      userKey.getPublicKey()
  );

  await masterAccount.createAccount(
    config.userAccount,
    userKey.getPublicKey(),
    new BN(10).pow(new BN(27))
  );
  const userAccount = new nearAPI.Account(near.connection, config.userAccount);
  const provider = near.connection.provider;

  console.log(
    "creating a second user account: " +
      config.user2Account +
      " with key " +
      user2Key.getPublicKey()
  );

  await masterAccount.createAccount(
    config.user2Account,
    user2Key.getPublicKey(),
    new BN(10).pow(new BN(27))
  );
  const user2Account = new nearAPI.Account(
    near.connection,
    config.user2Account
  );

  let token_bridge = CONTRACTS.DEVNET.near.token_bridge;
  let core_bridge = CONTRACTS.DEVNET.near.core;

  console.log("Setting up algorand wallet");

  let algoCore = BigInt(CONTRACTS.DEVNET.algorand.core);
  let algoToken = BigInt(CONTRACTS.DEVNET.algorand.token_bridge);

  const algoClient: algosdk.Algodv2 = getAlgoClient();
  const tempAccts: Account[] = await getTempAccounts();

  const algoWallet: Account = tempAccts[0];

  console.log("Creating USDC on Near");

  let ts = new TestLib();
  let seq = Math.floor(new Date().getTime() / 1000);
  let usdcvaa = ts.hexStringToUint8Array(
    ts.genAssetMeta(
      ts.singleGuardianPrivKey,
      0,
      1,
      seq,
      "4523c3F29447d1f32AEa95BEBD00383c4640F1b4".toLowerCase(),
      1,
      8,
      "USDC",
      "CircleCoin"
    )
  );

  seq = seq + 1;

  const createWrappedMsgs = await createWrappedOnNear(
    provider,
    token_bridge,
    usdcvaa
  );
  let usdc;
  for (const msg of createWrappedMsgs) {
    const tx = await userAccount.functionCall(msg);
    usdc = nearAPI.providers.getTransactionLastResult(tx);
  }
  console.log(usdc);

  console.log("Creating USDC token on algorand");
  let tx = await createWrappedOnAlgorand(
    algoClient,
    algoToken,
    algoCore,
    algoWallet.addr,
    usdcvaa
  );
  await signSendAndConfirmAlgorand(algoClient, tx, algoWallet);

  console.log("Registering the receiving account");

  let myAddress = nearAPI.providers.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: token_bridge,
      methodName: "register_account",
      args: { account: userAccount.accountId },
      gas: new BN("100000000000000"),
      attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
    })
  );
  console.log("myAddress: " + myAddress);

  console.log("Airdropping USDC on myself");
  {
    let trans = ts.genTransfer(
      ts.singleGuardianPrivKey,
      0,
      1,
      seq,
      10000,
      "4523c3F29447d1f32AEa95BEBD00383c4640F1b4".toLowerCase(),
      1,
      myAddress, // lets send it to the correct user (use the hash)
      CHAIN_ID_NEAR,
      0
    );
    console.log(trans);

    console.log(
      await redeemOnNear(
        provider,
        userAccount.account_id,
        token_bridge,
        hexToUint8Array(trans)
      )
    );
  }
  console.log(".. created some USDC on near");

  let wrappedTransfer;
  {
    console.log("transfer USDC from near to algorand");
    const transferMsg = await transferTokenFromNear(
      provider,
      userAccount.account_id,
      core_bridge,
      token_bridge,
      usdc,
      BigInt(100),
      decodeAddress(algoWallet.addr).publicKey,
      8,
      BigInt(0)
    );
    const transferOutcome = await userAccount.functionCall(transferMsg);
    const sequence = parseSequenceFromLogNear(transferOutcome);
    if (sequence === null) {
      console.log("sequence is null");
      return;
    }

    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      ["http://localhost:7071"],
      CHAIN_ID_NEAR,
      getEmitterAddressNear(token_bridge),
      sequence,
      {
        transport: NodeHttpTransport(),
      }
    );

    console.log("vaa: " + Buffer.from(signedVAA).toString("hex"));
    wrappedTransfer = signedVAA;
  }

  let usdcAssetId;
  {
    console.log("redeeming our wrapped USDC from Near on Algorand");
    const tx = await redeemOnAlgorand(
      algoClient,
      algoToken,
      algoCore,
      wrappedTransfer,
      algoWallet.addr
    );
    await signSendAndConfirmAlgorand(algoClient, tx, algoWallet);

    let p = _parseVAAAlgorand(wrappedTransfer);
    usdcAssetId = (await getForeignAssetAlgorand(
      algoClient,
      algoToken,
      p.FromChain as ChainId,
      p.Contract as string
    )) as bigint;
    console.log("usdc asset id: " + usdcAssetId);
  }

  const emitterAddr = getEmitterAddressAlgorand(algoToken);

  console.log("wallet addr: " + algoWallet.addr);
  console.log("usdcAssetId: " + usdcAssetId);

  console.log("transferring USDC from Algo To Near... getting the vaa");
  console.log("myAddress: " + myAddress);

  let testAddress = nearAPI.providers.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: token_bridge,
      methodName: "register_account",
      args: { account: "test.test.near" },
      gas: new BN("100000000000000"),
      attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
    })
  );
  console.log("testAddress: " + testAddress);

  let transferAlgoToNearP3;
  {
    const AmountToTransfer: number = 100;
    const Fee: number = 20;
    const transferTxs = await transferFromAlgorand(
      algoClient,
      algoToken,
      algoCore,
      algoWallet.addr,
      usdcAssetId,
      BigInt(AmountToTransfer),
      testAddress,
      CHAIN_ID_NEAR,
      BigInt(Fee),
      hexToUint8Array("ff")
    );
    const transferResult = await signSendAndConfirmAlgorand(
      algoClient,
      transferTxs,
      algoWallet
    );
    const txSid = parseSequenceFromLogAlgorand(transferResult);
    transferAlgoToNearP3 = (
      await getSignedVAAWithRetry(
        ["http://localhost:7071"],
        CHAIN_ID_ALGORAND,
        emitterAddr,
        txSid,
        { transport: NodeHttpTransport() }
      )
    ).vaaBytes;
  }

  console.log("redeeming P3 NEAR on Near");
  console.log(
    await redeemOnNear(
      provider,
      user2Account.account_id,
      token_bridge,
      transferAlgoToNearP3
    )
  );

  console.log("transferring USDC from Algo To Near... getting the vaa");
  let transferAlgoToNearRandoP3;
  {
    const AmountToTransfer: number = 100;
    const Fee: number = 20;
    const transferTxs = await transferFromAlgorand(
      algoClient,
      algoToken,
      algoCore,
      algoWallet.addr,
      usdcAssetId,
      BigInt(AmountToTransfer),
      testAddress,
      CHAIN_ID_NEAR,
      BigInt(Fee),
      hexToUint8Array("ff")
    );
    const transferResult = await signSendAndConfirmAlgorand(
      algoClient,
      transferTxs,
      algoWallet
    );
    const txSid = parseSequenceFromLogAlgorand(transferResult);
    transferAlgoToNearRandoP3 = (
      await getSignedVAAWithRetry(
        ["http://localhost:7071"],
        CHAIN_ID_ALGORAND,
        emitterAddr,
        txSid,
        { transport: NodeHttpTransport() }
      )
    ).vaaBytes;
  }

  console.log("What next?");
}

testNearSDK();
