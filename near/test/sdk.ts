// npx pretty-quick

const fetch = require("node-fetch");
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

import {
  connect as nearConnect,
  keyStores as nearKeyStores,
  utils as nearUtils,
  Account as nearAccount,
  providers as nearProviders,
} from "@certusone/wormhole-sdk/node_modules/near-api-js";

const BN = require("bn.js");

import { TestLib } from "./testlib";

import algosdk, {
  Account,
  decodeAddress,
  getApplicationAddress,
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
  attestNearFromNear,
  attestTokenFromNear,
  attestFromAlgorand,
  createWrappedOnAlgorand,
  createWrappedOnNear,
  getEmitterAddressAlgorand,
  getEmitterAddressNear,
  getForeignAssetAlgorand,
  getForeignAssetNear,
  getIsTransferCompletedNear,
  getSignedVAAWithRetry,
  redeemOnAlgorand,
  redeemOnNear,
  transferFromAlgorand,
  transferNearFromNear,
  transferTokenFromNear,
} from "@certusone/wormhole-sdk/src";

import { parseSequenceFromLogAlgorand } from "@certusone/wormhole-sdk/lib/cjs/bridge";

import { _parseVAAAlgorand } from "@certusone/wormhole-sdk/lib/cjs/algorand";
import { parseSequenceFromLogNear } from "@certusone/wormhole-sdk/src";

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

export async function createAsset(
  aClient: algosdk.Algodv2,
  account: Account
): Promise<any> {
  const params = await aClient.getTransactionParams().do();
  const note = undefined; // arbitrary data to be stored in the transaction; here, none is stored
  // Asset creation specific parameters
  const addr = account.addr;
  // Whether user accounts will need to be unfrozen before transacting
  const defaultFrozen = false;
  // integer number of decimals for asset unit calculation
  const decimals = 10;
  // total number of this asset available for circulation
  const totalIssuance = 1000000;
  // Used to display asset units to user
  const unitName = "NORIUM";
  // Friendly name of the asset
  const assetName = "ChuckNorium";
  // Optional string pointing to a URL relating to the asset
  // const assetURL = "http://www.chucknorris.com";
  const assetURL = "";
  // Optional hash commitment of some sort relating to the asset. 32 character length.
  // const assetMetadataHash = "16efaa3924a6fd9d3a4824799a4ac65d";
  const assetMetadataHash = "";
  // The following parameters are the only ones
  // that can be changed, and they have to be changed
  // by the current manager
  // Specified address can change reserve, freeze, clawback, and manager
  const manager = account.addr;
  // Specified address is considered the asset reserve
  // (it has no special privileges, this is only informational)
  const reserve = account.addr;
  // Specified address can freeze or unfreeze user asset holdings
  const freeze = account.addr;
  // Specified address can revoke user asset holdings and send
  // them to other addresses
  const clawback = account.addr;

  // signing and sending "txn" allows "addr" to create an asset
  const txn = algosdk.makeAssetCreateTxnWithSuggestedParams(
    addr,
    note,
    totalIssuance,
    decimals,
    defaultFrozen,
    manager,
    reserve,
    freeze,
    clawback,
    unitName,
    assetName,
    assetURL,
    assetMetadataHash,
    params
  );

  const rawSignedTxn = txn.signTxn(account.sk);
  const tx = await aClient.sendRawTransaction(rawSignedTxn).do();

  // wait for transaction to be confirmed
  const ptx = await algosdk.waitForConfirmation(aClient, tx.txId, 4);
  // Get the new asset's information from the creator account
  const assetID: number = ptx["asset-index"];
  //Get the completed Transaction
  return assetID;
}

export function logNearGas(result: any, comment: string) {
  const { totalGasBurned, totalTokensBurned } = result.receipts_outcome.reduce(
    (acc: any, receipt: any) => {
      acc.totalGasBurned += receipt.outcome.gas_burnt;
      acc.totalTokensBurned += nearUtils.format.formatNearAmount(
        receipt.outcome.tokens_burnt
      );
      return acc;
    },
    {
      totalGasBurned: result.transaction_outcome.outcome.gas_burnt,
      totalTokensBurned: nearUtils.format.formatNearAmount(
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

  let masterKey = nearUtils.KeyPair.fromString(
    keyFile.secret_key || keyFile.private_key
  );

  let keyStore = new nearKeyStores.InMemoryKeyStore();
  keyStore.setKey(
    config.networkId as string,
    config.masterAccount as string,
    masterKey
  );

  let near = await nearConnect({
    headers: {},
    keyStore,
    networkId: config.networkId as string,
    nodeUrl: config.nodeUrl as string,
  });
  let masterAccount = new nearAccount(
    near.connection,
    config.masterAccount as string
  );

  console.log(
    "Finish init NEAR masterAccount: " +
      JSON.stringify(await masterAccount.getAccountBalance())
  );

  let userKey = nearUtils.KeyPair.fromRandom("ed25519");
  keyStore.setKey(
    config.networkId as string,
    config.userAccount as string,
    userKey
  );
  let user2Key = nearUtils.KeyPair.fromRandom("ed25519");
  keyStore.setKey(
    config.networkId as string,
    config.user2Account as string,
    user2Key
  );

  console.log(
    "creating a user account: " +
      config.userAccount +
      " with key " +
      userKey.getPublicKey()
  );

  await masterAccount.createAccount(
    config.userAccount as string,
    userKey.getPublicKey(),
    new BN(10).pow(new BN(27))
  );
  const userAccount = new nearAccount(
    near.connection,
    config.userAccount as string
  );
  const provider = near.connection.provider;

  console.log(
    "creating a second user account: " +
      config.user2Account +
      " with key " +
      user2Key.getPublicKey()
  );

  await masterAccount.createAccount(
    config.user2Account as string,
    user2Key.getPublicKey(),
    new BN(10).pow(new BN(27))
  );
  const user2Account = new nearAccount(
    near.connection,
    config.user2Account as string
  );

  console.log(
    "Creating new random non-wormhole token and air dropping some tokens to myself"
  );

  let randoToken = nearProviders.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: "test.test.near",
      methodName: "deploy_ft",
      args: {
        account: userAccount.accountId,
      },
      gas: new BN("300000000000000"),
    })
  );

  let token_bridge = CONTRACTS.DEVNET.near.token_bridge;
  let core_bridge = CONTRACTS.DEVNET.near.core;

  console.log("Setting up algorand wallet");

  let algoCore = BigInt(CONTRACTS.DEVNET.algorand.core);
  let algoToken = BigInt(CONTRACTS.DEVNET.algorand.token_bridge);

  const tbAddr: string = getApplicationAddress(algoToken);
  const decTbAddr: Uint8Array = decodeAddress(tbAddr).publicKey;

  const algoClient: algosdk.Algodv2 = getAlgoClient();
  const tempAccts: Account[] = await getTempAccounts();

  const algoWallet: Account = tempAccts[0];

  let norium = await createAsset(algoClient, algoWallet);
  console.log("Norum asset-id on algorand", norium);

  const attestTxs = await attestFromAlgorand(
    algoClient,
    algoToken,
    algoCore,
    algoWallet.addr,
    BigInt(norium)
  );
  const attestResult = await signSendAndConfirmAlgorand(
    algoClient,
    attestTxs,
    algoWallet
  );

  const attestSn = parseSequenceFromLogAlgorand(attestResult);

  const emitterAddr = getEmitterAddressAlgorand(algoToken);

  const { vaaBytes } = await getSignedVAAWithRetry(
    ["http://localhost:7071"],
    CHAIN_ID_ALGORAND,
    emitterAddr,
    attestSn,
    { transport: NodeHttpTransport() }
  );

  for (const msg of await createWrappedOnNear(
    provider,
    token_bridge,
    vaaBytes
  )) {
    await userAccount.functionCall(msg);
  }

  console.log("for norium, createWrappedOnNear returned");

  let account_hash = await userAccount.viewFunction({
    contractId: token_bridge,
    methodName: "hash_account",
    args: {
      account: userAccount.accountId,
    },
  });

  console.log(account_hash);

  let myAddress = account_hash[1];

  // Start transfer from Algorand to Near
  console.log("Lets send 12300 Norum to near");
  const AmountToTransfer: number = 12300;
  const Fee: number = 0;
  const transferTxs = await transferFromAlgorand(
    algoClient,
    algoToken,
    algoCore,
    algoWallet.addr,
    BigInt(norium),
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
  const signedVaa = await getSignedVAAWithRetry(
    ["http://localhost:7071"],
    CHAIN_ID_ALGORAND,
    emitterAddr,
    txSid,
    { transport: NodeHttpTransport() }
  );

  console.log("Lets send 5123 ALGO to near");

  const ALGOTxs = await transferFromAlgorand(
    algoClient,
    algoToken,
    algoCore,
    algoWallet.addr,
    BigInt(0),
    BigInt(5123),
    myAddress,
    CHAIN_ID_NEAR,
    BigInt(Fee)
  );
  const ALGOResult = await signSendAndConfirmAlgorand(
    algoClient,
    ALGOTxs,
    algoWallet
  );
  const ALGOSid = parseSequenceFromLogAlgorand(ALGOResult);
  const ALGOVaa = await getSignedVAAWithRetry(
    ["http://localhost:7071"],
    CHAIN_ID_ALGORAND,
    emitterAddr,
    ALGOSid,
    { transport: NodeHttpTransport() }
  );

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
    (await getIsTransferCompletedNear(provider, token_bridge, usdcvaa)) == true
  ) {
    console.log("getIsTransferCompleted returned incorrect value (true)");
    process.exit(1);
  }

  const createWrappedMsgs = await createWrappedOnNear(
    provider,
    token_bridge,
    usdcvaa
  );
  let usdc;
  for (const msg of createWrappedMsgs) {
    const tx = await userAccount.functionCall(msg);
    usdc = nearProviders.getTransactionLastResult(tx);
  }
  console.log("createWrappedOnNear returned " + usdc);

  if (usdc === "") {
    console.log("null usdc ... we failed to create it?!");
    process.exit(1);
  }

  if (
    (await getIsTransferCompletedNear(provider, token_bridge, usdcvaa)) == false
  ) {
    console.log("getIsTransferCompleted returned incorrect value (false)");
    process.exit(1);
  }

  let aname = await getForeignAssetNear(
    provider,
    token_bridge,
    usdcp.FromChain as ChainId,
    usdcp.Contract as string
  );
  if (aname !== usdc) {
    console.log(aname + " !== " + usdc);
    process.exit(1);
  } else {
    console.log(aname + " === " + usdc);
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

    try {
      const redeemMsgs = await redeemOnNear(
        provider,
        userAccount.accountId,
        token_bridge,
        hexToUint8Array(trans)
      );
      for (const msg of redeemMsgs) {
        await userAccount.functionCall(msg);
      }
      console.log("This should have thrown a exception..");
      process.exit(1);
    } catch (error) {
      console.log("Exception thrown.. nice.. we dont suck");
      console.log(error);
    }

    console.log("Registering the receiving account");

    let myAddress2 = nearProviders.getTransactionLastResult(
      await userAccount.functionCall({
        contractId: token_bridge,
        methodName: "register_account",
        args: { account: userAccount.accountId },
        gas: new BN("100000000000000"),
        attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
      })
    );
    console.log("myAddress: " + myAddress2);

    const redeemMsgs = await redeemOnNear(
      provider,
      userAccount.accountId,
      token_bridge,
      hexToUint8Array(trans)
    );
    for (const msg of redeemMsgs) {
      await userAccount.functionCall(msg);
    }
  }
  console.log(".. created some USDC");

  console.log("Redeeming norium on near");
  for (const msg of await redeemOnNear(
    provider,
    userAccount.accountId,
    token_bridge,
    signedVaa.vaaBytes
  )) {
    await userAccount.functionCall(msg);
  }

  let nativeAttest;
  {
    console.log("attesting: " + randoToken);
    const attestMsgs = await attestTokenFromNear(
      provider,
      core_bridge,
      token_bridge,
      randoToken
    );
    let sequence;
    for (const msg of attestMsgs) {
      const tx = await userAccount.functionCall(msg);
      sequence = parseSequenceFromLogNear(tx);
    }
    if (!sequence) {
      console.log("sequence is null");
      process.exit(1);
    }
    console.log(getEmitterAddressNear(token_bridge));
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
    let p = _parseVAAAlgorand(signedVAA);

    console.log(p.FromChain as ChainId, p.Contract as string);

    let a = await getForeignAssetNear(
      provider,
      token_bridge,
      p.FromChain as ChainId,
      p.Contract as string
    );
    if (a !== randoToken) {
      console.log(a + " !== " + randoToken);
      process.exit(1);
    }

    nativeAttest = signedVAA;
  }

  let nearAttest;
  {
    console.log("attesting Near itself");
    const attestMsg = await attestNearFromNear(
      provider,
      core_bridge,
      token_bridge
    );
    const tx = await userAccount.functionCall(attestMsg);
    const sequence = parseSequenceFromLogNear(tx);
    if (sequence === null) {
      console.log("sequence is null");
      process.exit(1);
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
    let p = _parseVAAAlgorand(signedVAA);
    let a = await getForeignAssetNear(
      provider,
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

  if (usdc === "") {
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

    const transferMsgs = await transferTokenFromNear(
      provider,
      userAccount.accountId,
      core_bridge,
      token_bridge,
      usdc,
      BigInt(100),
      decodeAddress(algoWallet.addr).publicKey,
      8,
      BigInt(0)
    );
    let sequence;
    for (const msg of transferMsgs) {
      const tx = await userAccount.functionCall(msg);
      sequence = parseSequenceFromLogNear(tx);
    }
    if (!sequence) {
      console.log("sequence is null");
      process.exit(1);
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

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: userAccount.accountId,
      },
    })
  );

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: token_bridge,
      },
    })
  );

  let randoTransfer;
  {
    console.log("YYY transfer rando token from near to algorand");
    const transferMsgs = await transferTokenFromNear(
      provider,
      userAccount.accountId,
      core_bridge,
      token_bridge,
      randoToken,
      BigInt(21) * BigInt("1000000000") * BigInt("1000000000000000000"),
      decodeAddress(algoWallet.addr).publicKey,
      8,
      BigInt(0)
    );
    let sequence;
    for (const msg of transferMsgs) {
      const tx = await userAccount.functionCall(msg);
      sequence = parseSequenceFromLogNear(tx);
    }
    if (!sequence) {
      console.log("sequence is null");
      process.exit(1);
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
    randoTransfer = signedVAA;

    console.log(_parseVAAAlgorand(randoTransfer));
  }

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: userAccount.accountId,
      },
    })
  );

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: token_bridge,
      },
    })
  );

  let nearTransfer;
  {
    console.log("transfer near from near to algorand");
    const transferMsg = await transferNearFromNear(
      provider,
      core_bridge,
      token_bridge,
      BigInt(1000000000000000000000000),
      decodeAddress(algoWallet.addr).publicKey,
      8,
      BigInt(0)
    );
    const tx = await userAccount.functionCall(transferMsg);
    const sequence = parseSequenceFromLogNear(tx);
    if (sequence === null) {
      console.log("sequence is null");
      process.exit(1);
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
    console.log("YYY redeeming our near native asset on Algorand");
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

  console.log("wallet addr: " + algoWallet.addr);
  console.log("usdcAssetId: " + usdcAssetId);

  console.log("transferring USDC from Algo To Near... getting the vaa");
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

  console.log("YYY transferring rando from Algo To Near... getting the vaa");
  let transferAlgoToNearRando;
  {
    const Fee: number = 0;
    const transferTxs = await transferFromAlgorand(
      algoClient,
      algoToken,
      algoCore,
      algoWallet.addr,
      randoAssetId,
      BigInt(20) * BigInt("1000000000") * BigInt("100000000"),
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

  console.log("transferring NEAR from Algo To Near... getting the vaa");
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
  let redeemMsgs = await redeemOnNear(
    provider,
    user2Account.accountId,
    token_bridge,
    transferAlgoToNearUSDC
  );
  for (const msg of redeemMsgs) {
    await userAccount.functionCall(msg);
  }

  console.log(
    "YYY redeeming Rando on Near: " + uint8ArrayToHex(transferAlgoToNearRando)
  );

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: userAccount.accountId,
      },
    })
  );

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: token_bridge,
      },
    })
  );

  redeemMsgs = await redeemOnNear(
    provider,
    user2Account.accountId,
    token_bridge,
    transferAlgoToNearRando
  );
  for (const msg of redeemMsgs) {
    await userAccount.functionCall(msg);
  }

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: userAccount.accountId,
      },
    })
  );

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: token_bridge,
      },
    })
  );

  console.log("redeeming NEAR on Near");

  redeemMsgs = await redeemOnNear(
    provider,
    user2Account.accountId,
    token_bridge,
    transferAlgoToNearNEAR
  );
  for (const msg of redeemMsgs) {
    await userAccount.functionCall(msg);
  }

  let userAccount2Address = nearProviders.getTransactionLastResult(
    await userAccount.functionCall({
      contractId: token_bridge,
      methodName: "register_account",
      args: { account: user2Account.accountId },
      gas: new BN("100000000000000"),
      attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
    })
  );
  console.log("userAccount2Address: " + userAccount2Address);

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
      userAccount2Address,
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

  {
    console.log("redeeming P3 NEAR on Near");
    console.log(
      await redeemOnNear(
        provider,
        user2Account.accountId,
        token_bridge,
        transferAlgoToNearP3
      )
    );

    console.log(
      "YYY P3 transferring rando from Algo To Near... getting the vaa"
    );
    let transferAlgoToNearRandoP3;
    {
      const Fee: number = 0;
      const transferTxs = await transferFromAlgorand(
        algoClient,
        algoToken,
        algoCore,
        algoWallet.addr,
        randoAssetId,
        BigInt(1) * BigInt("1000000000") * BigInt("100000000"),
        userAccount2Address,
        CHAIN_ID_NEAR,
        BigInt(Fee) * BigInt("100000000"),
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

    console.log("YYY redeeming P3 random on Near");
    const redeemMsgs = await redeemOnNear(
      provider,
      user2Account.accountId,
      token_bridge,
      transferAlgoToNearRandoP3
    );
    for (const msg of redeemMsgs) {
      await user2Account.functionCall(msg);
    }
  }

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: userAccount.accountId,
      },
    })
  );

  console.log(
    await userAccount.viewFunction({
      contractId: randoToken,
      methodName: "ft_balance_of",
      args: {
        account_id: token_bridge,
      },
    })
  );

  console.log("What next?");
}

testNearSDK();
