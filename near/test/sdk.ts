// npx pretty-quick

const sha256 = require("js-sha256");
const nearAPI = require("near-api-js");
const fs = require("fs").promises;
const assert = require("assert").strict;
const fetch = require("node-fetch");
const elliptic = require("elliptic");
const web3Utils = require("web3-utils");
import { zeroPad } from "@ethersproject/bytes";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

import { Account as nearAccount } from "@certusone/wormhole-sdk/node_modules/near-api-js";
const BN = require("bn.js");

import { TestLib } from "./testlib";

import algosdk, {
  Account,
  Algodv2,
  OnApplicationComplete,
  SuggestedParams,
  bigIntToBytes,
  decodeAddress,
  getApplicationAddress,
  makeApplicationCallTxnFromObject,
  makePaymentTxnWithSuggestedParamsFromObject,
  waitForConfirmation,
} from "@certusone/wormhole-sdk/node_modules/algosdk";

import {
  createAsset,
  getAlgoClient,
  getBalance,
  getBalances,
  getForeignAssetFromVaaAlgorand,
  getTempAccounts,
  signSendAndConfirmAlgorand,
} from "./algoHelpers";

import {
  CHAIN_ID_ALGORAND,
  CHAIN_ID_NEAR,
  ChainId,
  ChainName,
  textToHexString,
  textToUint8Array,
} from "@certusone/wormhole-sdk/lib/cjs/utils";

import { safeBigIntToNumber } from "@certusone/wormhole-sdk/lib/cjs/utils/bigint";

import {
  CONTRACTS,
  attestNearFromNear,
  attestTokenFromNear,
  attestFromAlgorand,
  createWrappedOnAlgorand,
  createWrappedOnNear,
  getEmitterAddressAlgorand,
  getForeignAssetAlgorand,
  getForeignAssetNear,
  getIsTransferCompletedNear,
  getIsWrappedAssetNear,
  getOriginalAssetNear,
  getSignedVAAWithRetry,
  redeemOnAlgorand,
  redeemOnNear,
  transferFromAlgorand,
  transferNearFromNear,
  transferTokenFromNear,
} from "@certusone/wormhole-sdk";

const wh = require("@certusone/wormhole-sdk");

import { parseSequenceFromLogAlgorand } from "@certusone/wormhole-sdk/lib/cjs/bridge";

import {
  getMessageFee,
  optin,
  TransactionSignerPair,
  _parseVAAAlgorand,
} from "@certusone/wormhole-sdk/lib/cjs/algorand";

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

export function parseSequenceFromLogNear(result: any): [number, string] {
  let sequence = "";
  for (const o of result.receipts_outcome) {
    for (const l of o.outcome.logs) {
      if (l.startsWith("EVENT_JSON:")) {
        const body = JSON.parse(l.slice(11));
        if (body.standard == "wormhole" && body.event == "publish") {
          return [body.seq, body.emitter];
        }
      }
    }
  }
  return [-1, ""];
}

async function testNearSDK() {
  let config = getConfig(process.env.NEAR_ENV || "sandbox");

  // Retrieve the validator key directly in the Tilt environment
  const response = await fetch("http://localhost:3031/validator_key.json");

  const keyFile = await response.json();

  let masterKey = nearAPI.utils.KeyPair.fromString(
    keyFile.secret_key || keyFile.private_key
  );
  let masterPubKey = masterKey.getPublicKey();

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

  console.log(
    "Creating new random non-wormhole token and air dropping some tokens to myself"
  );

  let randoToken = nearAPI.providers.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: "test.test.near",
      methodName: "deploy_ft",
      args: {
        account: userAccount.accountId,
      },
      gas: 300000000000000,
    })
  );

  let token_bridge = CONTRACTS.DEVNET.near.token_bridge;
  let core_bridge = CONTRACTS.DEVNET.near.core;

  console.log("Setting up algorand wallet");

  let algoCore = BigInt(CONTRACTS.DEVNET.algorand.core);
  let algoToken = BigInt(CONTRACTS.DEVNET.algorand.token_bridge);

  const tbAddr: string = getApplicationAddress(algoToken);
  const decTbAddr: Uint8Array = decodeAddress(tbAddr).publicKey;
  const aa: string = uint8ArrayToHex(decTbAddr);

  const algoClient: algosdk.Algodv2 = getAlgoClient();
  const tempAccts: Account[] = await getTempAccounts();
  const numAccts: number = tempAccts.length;

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

  let usdcp = _parseVAAAlgorand(usdcvaa);

  //console.log(usdcp);

  console.log("calling createWrappedOnNear to create usdc");

  if (
    (await getIsTransferCompletedNear(userAccount, token_bridge, usdcvaa)) ==
    true
  ) {
    console.log("getIsTransferCompleted returned incorrect value (true)");
    process.exit(1);
  }

  let usdc = await createWrappedOnNear(userAccount, token_bridge, usdcvaa);
  console.log(usdc);

  if (usdc == "") {
    console.log("null usdc ... we failed to create it?!");
    process.exit(1);
  }

  if (
    (await getIsTransferCompletedNear(userAccount, token_bridge, usdcvaa)) ==
    false
  ) {
    console.log("getIsTransferCompleted returned incorrect value (false)");
    process.exit(1);
  }

  let aname = await getForeignAssetNear(
    userAccount,
    token_bridge,
    usdcp.FromChain as ChainId,
    usdcp.Contract as string
  );
  if (aname != usdc) {
    console.log(aname + " != " + usdc);
    process.exit(1);
  } else {
    console.log(aname + " == " + usdc);
  }

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
      await redeemOnNear(userAccount, token_bridge, hexToUint8Array(trans))
    );
  }
  console.log(".. created some USDC");

  let nativeAttest;
  {
    console.log("attesting: " + randoToken);
    let s = await attestTokenFromNear(
      userAccount,
      core_bridge,
      token_bridge,
      randoToken
    );
    console.log(s);
    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      ["http://localhost:7071"],
      CHAIN_ID_NEAR,
      s[1],
      s[0].toString(),
      {
        transport: NodeHttpTransport(),
      }
    );

    console.log("vaa: " + Buffer.from(signedVAA).toString("hex"));
    let p = _parseVAAAlgorand(signedVAA);

    console.log(p.FromChain as ChainId, p.Contract as string);

    let a = await getForeignAssetNear(
      userAccount,
      token_bridge,
      p.FromChain as ChainId,
      p.Contract as string
    );
    if (a != randoToken) {
      console.log(a + " != " + randoToken);
      process.exit(1);
    }

    nativeAttest = signedVAA;
  }

  let nearAttest;
  {
    console.log("attesting Near itself");
    let s = await attestNearFromNear(userAccount, core_bridge, token_bridge);

    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      ["http://localhost:7071"],
      CHAIN_ID_NEAR,
      s[1],
      s[0].toString(),
      {
        transport: NodeHttpTransport(),
      }
    );

    console.log("vaa: " + Buffer.from(signedVAA).toString("hex"));
    let p = _parseVAAAlgorand(signedVAA);
    let a = await getForeignAssetNear(
      userAccount,
      token_bridge,
      p.FromChain as ChainId,
      p.Contract as string
    );
    console.log(
      "chain: {}   contract: {}   account: {}",
      p.FromChain,
      p.Contract,
      a
    );

    nearAttest = signedVAA;
  }

  console.log("Creating a native token from near onto algorand");
  tx = await createWrappedOnAlgorand(
    algoClient,
    algoToken,
    algoCore,
    algoWallet.addr,
    nativeAttest
  );
  await signSendAndConfirmAlgorand(algoClient, tx, algoWallet);
  console.log("Creating NEAR from near onto algorand");
  tx = await createWrappedOnAlgorand(
    algoClient,
    algoToken,
    algoCore,
    algoWallet.addr,
    nearAttest
  );
  await signSendAndConfirmAlgorand(algoClient, tx, algoWallet);

  console.log("Shock and awe...");

  if (usdc == "") {
    console.log("null usdc");
    process.exit(1);
  }

  let wrappedTransfer;
  {
    console.log(
      "transfer wrapped token from near to algorand",
      userAccount,
      core_bridge,
      token_bridge,
      usdc
    );

    let s = await transferTokenFromNear(
      userAccount,
      core_bridge,
      token_bridge,
      usdc,
      BigInt(100),
      decodeAddress(algoWallet.addr).publicKey,
      8,
      BigInt(0)
    );

    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      ["http://localhost:7071"],
      CHAIN_ID_NEAR,
      s[1],
      s[0].toString(),
      {
        transport: NodeHttpTransport(),
      }
    );

    console.log("vaa: " + Buffer.from(signedVAA).toString("hex"));
    wrappedTransfer = signedVAA;
  }

  let randoTransfer;
  {
    console.log("transfer rando token from near to algorand");
    let s = await transferTokenFromNear(
      userAccount,
      core_bridge,
      token_bridge,
      randoToken,
      BigInt(10000000),
      decodeAddress(algoWallet.addr).publicKey,
      8,
      BigInt(0)
    );

    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      ["http://localhost:7071"],
      CHAIN_ID_NEAR,
      s[1],
      s[0].toString(),
      {
        transport: NodeHttpTransport(),
      }
    );

    console.log("vaa: " + Buffer.from(signedVAA).toString("hex"));
    randoTransfer = signedVAA;
  }

  let nearTransfer;
  {
    console.log("transfer near from near to algorand");
    let s = await transferNearFromNear(
      userAccount,
      core_bridge,
      token_bridge,
      BigInt(1000000000000000000000000),
      decodeAddress(algoWallet.addr).publicKey,
      8,
      BigInt(0)
    );

    const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
      ["http://localhost:7071"],
      CHAIN_ID_NEAR,
      s[1],
      s[0].toString(),
      {
        transport: NodeHttpTransport(),
      }
    );

    console.log("vaa: " + Buffer.from(signedVAA).toString("hex"));
    nearTransfer = signedVAA;
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

  let randoAssetId;
  {
    console.log("redeeming our near native asset on Algorand");
    const tx = await redeemOnAlgorand(
      algoClient,
      algoToken,
      algoCore,
      randoTransfer,
      algoWallet.addr
    );
    await signSendAndConfirmAlgorand(algoClient, tx, algoWallet);

    let p = _parseVAAAlgorand(randoTransfer);
    randoAssetId = (await getForeignAssetAlgorand(
      algoClient,
      algoToken,
      p.FromChain as ChainId,
      p.Contract as string
    )) as bigint;
    console.log("randoToken asset id: " + randoAssetId);
  }

  let nearAssetId;
  {
    console.log("redeeming NEAR on Algorand");
    const tx = await redeemOnAlgorand(
      algoClient,
      algoToken,
      algoCore,
      nearTransfer,
      algoWallet.addr
    );
    await signSendAndConfirmAlgorand(algoClient, tx, algoWallet);

    let p = _parseVAAAlgorand(nearTransfer);
    nearAssetId = (await getForeignAssetAlgorand(
      algoClient,
      algoToken,
      p.FromChain as ChainId,
      p.Contract as string
    )) as bigint;
    console.log("NEAR asset id: " + nearAssetId);
  }

  const emitterAddr = getEmitterAddressAlgorand(algoToken);

  console.log("wallet addr: " + algoWallet.addr);
  console.log("usdcAssetId: " + usdcAssetId);

  console.log("transfering USDC from Algo To Near... getting the vaa");
  console.log("myAddress: " + myAddress);

  let transferAlgoToNearUSDC;
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
      myAddress,
      CHAIN_ID_NEAR,
      BigInt(Fee)
    );
    const transferResult = await signSendAndConfirmAlgorand(
      algoClient,
      transferTxs,
      algoWallet
    );
    const txSid = parseSequenceFromLogAlgorand(transferResult);
    transferAlgoToNearUSDC = (
      await getSignedVAAWithRetry(
        ["http://localhost:7071"],
        CHAIN_ID_ALGORAND,
        emitterAddr,
        txSid,
        { transport: NodeHttpTransport() }
      )
    ).vaaBytes;
  }

  console.log("transfering rando from Algo To Near... getting the vaa");
  let transferAlgoToNearRando;
  {
    const AmountToTransfer: number = 100;
    const Fee: number = 20;
    const transferTxs = await transferFromAlgorand(
      algoClient,
      algoToken,
      algoCore,
      algoWallet.addr,
      randoAssetId,
      BigInt(AmountToTransfer),
      myAddress,
      CHAIN_ID_NEAR,
      BigInt(Fee)
    );
    const transferResult = await signSendAndConfirmAlgorand(
      algoClient,
      transferTxs,
      algoWallet
    );
    const txSid = parseSequenceFromLogAlgorand(transferResult);
    transferAlgoToNearRando = (
      await getSignedVAAWithRetry(
        ["http://localhost:7071"],
        CHAIN_ID_ALGORAND,
        emitterAddr,
        txSid,
        { transport: NodeHttpTransport() }
      )
    ).vaaBytes;
  }

  console.log("transfering NEAR from Algo To Near... getting the vaa");
  let transferAlgoToNearNEAR;
  {
    const AmountToTransfer: number = 100;
    const Fee: number = 20;
    const transferTxs = await transferFromAlgorand(
      algoClient,
      algoToken,
      algoCore,
      algoWallet.addr,
      nearAssetId,
      BigInt(AmountToTransfer),
      myAddress,
      CHAIN_ID_NEAR,
      BigInt(Fee)
    );
    const transferResult = await signSendAndConfirmAlgorand(
      algoClient,
      transferTxs,
      algoWallet
    );
    const txSid = parseSequenceFromLogAlgorand(transferResult);
    transferAlgoToNearNEAR = (
      await getSignedVAAWithRetry(
        ["http://localhost:7071"],
        CHAIN_ID_ALGORAND,
        emitterAddr,
        txSid,
        { transport: NodeHttpTransport() }
      )
    ).vaaBytes;
  }

  console.log("redeeming USDC on Near");
  console.log(
    await redeemOnNear(user2Account, token_bridge, transferAlgoToNearUSDC)
  );

  console.log(
    "redeeming Rando on Near: " + uint8ArrayToHex(transferAlgoToNearRando)
  );
  console.log(
    await redeemOnNear(user2Account, token_bridge, transferAlgoToNearRando)
  );

  console.log("redeeming NEAR on Near");
  console.log(
    await redeemOnNear(user2Account, token_bridge, transferAlgoToNearNEAR)
  );

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
      nearAssetId,
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

  if (false) {
    console.log("redeeming P3 NEAR on Near");
    console.log(
      await redeemOnNear(user2Account, token_bridge, transferAlgoToNearP3)
    );

    console.log("transfering rando from Algo To Near... getting the vaa");
    let transferAlgoToNearRandoP3;
    {
      const AmountToTransfer: number = 100;
      const Fee: number = 20;
      const transferTxs = await transferFromAlgorand(
        algoClient,
        algoToken,
        algoCore,
        algoWallet.addr,
        randoAssetId,
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

    console.log("redeeming P3 random on Near");
    console.log(
      await redeemOnNear(user2Account, token_bridge, transferAlgoToNearRandoP3)
    );
  }

  console.log("What next?");
}

testNearSDK();
