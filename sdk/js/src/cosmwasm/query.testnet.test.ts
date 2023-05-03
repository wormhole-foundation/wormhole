import { getNetworkInfo, Network } from "@injectivelabs/networks";
import { getStdFee, DEFAULT_STD_FEE } from "@injectivelabs/utils";
import {
  PrivateKey,
  TxGrpcApi,
  ChainGrpcWasmApi,
  ChainRestAuthApi,
  createTransaction,
  MsgArg,
} from "@injectivelabs/sdk-ts";
import { expect, test } from "@jest/globals";
import {
  attestFromAlgorand,
  attestFromInjective,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_BSC,
  CHAIN_ID_INJECTIVE,
  coalesceChainId,
  CONTRACTS,
  createWrappedOnAlgorand,
  createWrappedOnInjective,
  getEmitterAddressAlgorand,
  getEmitterAddressInjective,
  getForeignAssetInjective,
  getIsTransferCompletedAlgorand,
  getIsTransferCompletedInjective,
  getOriginalAssetInjective,
  getSignedVAAWithRetry,
  hexToUint8Array,
  parseSequenceFromLogAlgorand,
  parseSequenceFromLogInjective,
  redeemOnAlgorand,
  redeemOnInjective,
  safeBigIntToNumber,
  textToUint8Array,
  transferFromAlgorand,
  transferFromInjective,
  tryHexToNativeString,
  tryNativeToHexString,
  tryNativeToUint8Array,
  uint8ArrayToHex,
} from "..";
import { CLUSTER } from "../token_bridge/__tests__/utils/consts";
import algosdk, {
  Account,
  Algodv2,
  decodeAddress,
  makeApplicationCallTxnFromObject,
  mnemonicToSecretKey,
  OnApplicationComplete,
  waitForConfirmation,
} from "algosdk";
import {
  getBalances,
  getForeignAssetFromVaaAlgorand,
  signSendAndConfirmAlgorand,
} from "../algorand/__tests__/testHelpers";
import { _parseVAAAlgorand } from "../algorand";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { fromUint8Array } from "js-base64";

function getEndPoint() {
  return CLUSTER === "mainnet" ? Network.MainnetK8s : Network.TestnetK8s;
}

test.skip("testnet - injective contract is own admin", async () => {
  const network = getNetworkInfo(getEndPoint());
  const client = new ChainGrpcWasmApi(network.grpc);
  const coreQueryResult = await client.fetchContractInfo(
    CONTRACTS.TESTNET.injective.core
  );
  expect(coreQueryResult?.admin).toEqual(CONTRACTS.TESTNET.injective.core);
  const tbQueryResult = await client.fetchContractInfo(
    CONTRACTS.TESTNET.injective.token_bridge
  );
  expect(tbQueryResult?.admin).toEqual(
    CONTRACTS.TESTNET.injective.token_bridge
  );
});
test.skip("testnet - injective query guardian_set_info", async () => {
  const network = getNetworkInfo(getEndPoint());
  const client = new ChainGrpcWasmApi(network.grpc);
  // https://k8s.testnet.lcd.injective.network/cosmwasm/wasm/v1/contract/inj1xx3aupmgv3ce537c0yce8zzd3sz567syuyedpg/smart/eyJndWFyZGlhbl9zZXRfaW5mbyI6e319
  const queryResult = await client.fetchSmartContractState(
    CONTRACTS.TESTNET.injective.core,
    Buffer.from('{"guardian_set_info":{}}').toString("base64")
  );
  let result: any = null;
  if (typeof queryResult.data === "string") {
    result = JSON.parse(
      Buffer.from(queryResult.data, "base64").toString("utf-8")
    );
  }
  expect(result?.guardian_set_index).toEqual(0);
  expect(result?.addresses.length).toEqual(1);
});
test.skip("testnet - injective query state", async () => {
  const network = getNetworkInfo(getEndPoint());
  const client = new ChainGrpcWasmApi(network.grpc);
  const queryResult = await client.fetchSmartContractState(
    CONTRACTS.TESTNET.injective.core,
    Buffer.from('{"get_state":{}}').toString("base64")
  );
  let result: any = null;
  if (typeof queryResult.data === "string") {
    result = JSON.parse(
      Buffer.from(queryResult.data, "base64").toString("utf-8")
    );
  }
  expect(result?.fee?.denom).toEqual("inj");
  expect(result?.fee?.amount).toEqual("0");
});
test.skip("testnet - injective query token bridge", async () => {
  const network = getNetworkInfo(getEndPoint());
  const client = new ChainGrpcWasmApi(network.grpc);
  // const wrappedAsset = await getIsWrappedAssetInjective(
  //   CONTRACTS.TESTNET.injective.token_bridge,
  //   "inj10jc4vr9vfq0ykkmfvfgz430w8z6hwdlqhdw9l8"
  // );
  // console.log("isWrappedAsset", wrappedAsset);
  const queryResult = await client.fetchSmartContractState(
    CONTRACTS.TESTNET.injective.token_bridge,
    Buffer.from(
      JSON.stringify({
        wrapped_registry: {
          chain: CHAIN_ID_BSC,
          address: Buffer.from(
            "000000000000000000000000ae13d989dac2f0debff460ac112a837c89baa7cd",
            "hex"
          ).toString("base64"),
        },
      })
    ).toString("base64")
  );
  let result: any = null;
  if (typeof queryResult.data === "string") {
    result = JSON.parse(
      Buffer.from(queryResult.data, "base64").toString("utf-8")
    );
    console.log("result", result);
  }
});
test.skip("testnet - injective attest native asset", async () => {
  //  Local consts
  const tba = CONTRACTS.TESTNET.injective.token_bridge;
  const network = getNetworkInfo(getEndPoint());
  const client = new ChainGrpcWasmApi(network.grpc);

  // Set up Inj wallet
  const walletPKHash: string = process.env.ETH_KEY || "";
  const walletPK = PrivateKey.fromHex(walletPKHash);
  const walletInjAddr = walletPK.toBech32();
  const walletPublicKey = walletPK.toPublicKey().toBase64();
  const accountDetails = await new ChainRestAuthApi(network.rest).fetchAccount(
    walletInjAddr
  );

  // Attest native inj
  const result = await attestFromInjective(tba, walletInjAddr, "inj");
  console.log("token", JSON.stringify(result.params.exec));

  // Create the transaction
  console.log("creating transaction...");
  const { signBytes, txRaw } = createTransaction({
    message: result,
    memo: "",
    fee: getStdFee((parseInt(DEFAULT_STD_FEE.gas, 10) * 2.5).toString()),
    pubKey: walletPublicKey,
    sequence: parseInt(accountDetails.account.base_account.sequence, 10),
    accountNumber: parseInt(
      accountDetails.account.base_account.account_number,
      10
    ),
    chainId: network.chainId,
  });

  /** Sign transaction */
  const signature = await walletPK.sign(Buffer.from(signBytes));

  /** Append Signatures */
  txRaw.signatures = [signature];
  const txService = new TxGrpcApi(network.grpc);

  console.log("Simulating transaction...");
  /** Simulate transaction */
  const simulationResponse = await txService.simulate(txRaw);
  console.log(
    `Transaction simulation response: ${JSON.stringify(
      simulationResponse.gasInfo
    )}`
  );

  /** Broadcast transaction */
  const txResponse = await txService.broadcast(txRaw);
  console.log(
    `Broadcasted transaction hash: ${JSON.stringify(txResponse.txHash)}`
  );

  // Need to get the VAA and parse it.
  const logSeq: string = parseSequenceFromLogInjective(txResponse);
  console.log("logSeq:", logSeq);
  const emitterAddress = await getEmitterAddressInjective(
    CONTRACTS.TESTNET.injective.token_bridge
  );
  const rpc: string[] = ["https://wormhole-v2-testnet-api.certus.one"];
  const { vaaBytes: nativeAssetVaa } = await getSignedVAAWithRetry(
    rpc,
    CHAIN_ID_INJECTIVE,
    emitterAddress,
    logSeq,
    {
      transport: NodeHttpTransport(), //This should only be needed when running in node.
    },
    1000, //retryTimeout
    1000 //Maximum retry attempts
  );
  console.log("signed VAA", uint8ArrayToHex(nativeAssetVaa));
  const parsedVAA = _parseVAAAlgorand(nativeAssetVaa);
  console.log("parsed attestation vaa", parsedVAA);
  const assetIdFromVaa = parsedVAA.Contract || "";
  console.log("assetIdFromVaa:", assetIdFromVaa);

  // const origAsset = await getOriginalAssetInjective(
  //   assetIdFromVaa,
  //   // tryHexToNativeString(assetIdFromVaa, "injective"),
  //   client
  // );
  // console.log("origAsset:", origAsset);
  // const natString = tryHexToNativeString(
  //   uint8ArrayToHex(origAsset.assetAddress),
  //   "injective"
  // );
  // console.log("natString:", natString);
});
test.skip("testnet - injective attest foreign asset", async () => {
  const tba = CONTRACTS.TESTNET.injective.token_bridge;
  const wallet = "inj13un2qqjaenrvlsr605u82c5q5y8zjkkhdgcetq";
  const foreignAssetAddress = "inj13772jvadyx4j0hrlfh4jzk0v39k8uyfxrfs540";
  const result = await attestFromInjective(tba, wallet, foreignAssetAddress);
  console.log("token", JSON.stringify(result.params.exec));
  console.log("json", result.toJSON());
  const walletPKHash = process.env.ETH_KEY || "";
  const walletPK = PrivateKey.fromPrivateKey(walletPKHash);
  const walletInjAddr = walletPK.toBech32();
  const walletPublicKey = walletPK.toPublicKey().toBase64();

  const network = getNetworkInfo(getEndPoint());
  /** Account Details **/
  const accountDetails = await new ChainRestAuthApi(network.rest).fetchAccount(
    walletInjAddr
  );
  const { signBytes, txRaw } = createTransaction({
    message: result,
    memo: "",
    fee: getStdFee((parseInt(DEFAULT_STD_FEE.gas, 10) * 2.5).toString()),
    pubKey: walletPublicKey,
    sequence: parseInt(accountDetails.account.base_account.sequence, 10),
    accountNumber: parseInt(
      accountDetails.account.base_account.account_number,
      10
    ),
    chainId: network.chainId,
  });
  /** Sign transaction */
  const signature = await walletPK.sign(Buffer.from(signBytes));

  /** Append Signatures */
  txRaw.signatures = [signature];
  const txService = new TxGrpcApi(network.grpc);

  /** Simulate transaction */
  const simulationResponse = await txService.simulate(txRaw);
  console.log(
    `Transaction simulation response: ${JSON.stringify(
      simulationResponse.gasInfo
    )}`
  );

  /** Broadcast transaction */
  const txResponse = await txService.broadcast(txRaw);
  console.log(
    `Broadcasted transaction hash: ${JSON.stringify(txResponse.txHash)}`
  );

  // expect(result?.fee?.denom).toEqual("inj");
  // expect(result?.fee?.amount).toEqual("0");
});
test.skip("testnet - injective get foreign asset", async () => {
  const tba = CONTRACTS.TESTNET.injective.token_bridge;
  const network = getNetworkInfo(getEndPoint());
  const client = new ChainGrpcWasmApi(network.grpc);
  // const foreignAssetAddress = "inj10jc4vr9vfq0ykkmfvfgz430w8z6hwdlqhdw9l8";
  const foreignAssetAddress =
    "000000000000000000000000ae13d989dac2f0debff460ac112a837c89baa7cd";
  const result = await getForeignAssetInjective(
    tba,
    client,
    CHAIN_ID_BSC,
    hexToUint8Array(foreignAssetAddress)
  );
  console.log("result", result);

  expect(result?.length).toBeGreaterThan(0);
});
// TODO: fix ALGO_MNEMONIC
test.skip("testnet - injective submit a vaa", async () => {
  try {
    // Set up Algorand side
    const algodToken = "";
    const algodServer = "https://testnet-api.algonode.cloud";
    const algodPort = "";
    const algodClient = new Algodv2(algodToken, algodServer, algodPort);

    console.log("Doing Algorand part......");
    console.log("Creating wallet...");
    const algoWallet: Account = mnemonicToSecretKey(
      process.env.ALGO_MNEMONIC || ""
    );

    console.log("wallet", algoWallet);

    const accountInfo = await algodClient
      .accountInformation(algoWallet.addr)
      .do();
    console.log("accountInfo", accountInfo);

    // Attest native ALGO on Algorand
    // Asset Index of native ALGO is 0
    const AlgoIndex = BigInt(0);
    const CoreID = BigInt(86525623); // Testnet
    const TokenBridgeID = BigInt(86525641); // Testnet
    const b = await getBalances(algodClient, algoWallet.addr);
    console.log("balances", b);
    const txs = await attestFromAlgorand(
      algodClient,
      TokenBridgeID,
      CoreID,
      algoWallet.addr,
      AlgoIndex
    );
    console.log("txs", txs);

    const result = await signSendAndConfirmAlgorand(
      algodClient,
      txs,
      algoWallet
    );
    console.log("result", result);

    const sn = parseSequenceFromLogAlgorand(result);
    console.log("sn", sn);

    // Now, try to send a NOP
    const suggParams: algosdk.SuggestedParams = await algodClient
      .getTransactionParams()
      .do();
    const nopTxn = makeApplicationCallTxnFromObject({
      from: algoWallet.addr,
      appIndex: safeBigIntToNumber(TokenBridgeID),
      onComplete: OnApplicationComplete.NoOpOC,
      appArgs: [textToUint8Array("nop")],
      suggestedParams: suggParams,
    });
    const resp = await algodClient
      .sendRawTransaction(nopTxn.signTxn(algoWallet.sk))
      .do();
    await waitForConfirmation(algodClient, resp.txId, 4);
    // End of NOP

    // Attestation on Algorand is complete.  Get the VAA
    // Guardian part
    const rpc: string[] = ["https://wormhole-v2-testnet-api.certus.one"];
    const emitterAddr = getEmitterAddressAlgorand(BigInt(TokenBridgeID));
    const { vaaBytes } = await getSignedVAAWithRetry(
      rpc,
      CHAIN_ID_ALGORAND,
      emitterAddr,
      sn,
      { transport: NodeHttpTransport() }
    );
    const pvaa = _parseVAAAlgorand(vaaBytes);
    console.log("parsed vaa", pvaa);

    // Submit the VAA on the Injective side
    // Start of Injective side
    console.log("Start doing the Injective part......");
    const tba = CONTRACTS.TESTNET.injective.token_bridge;
    const walletPKHash = process.env.ETH_KEY || "";
    const walletPK = PrivateKey.fromPrivateKey(walletPKHash);
    const walletInjAddr = walletPK.toBech32();
    const walletPublicKey = walletPK.toPublicKey().toBase64();

    const network = getNetworkInfo(getEndPoint());
    const client = new ChainGrpcWasmApi(network.grpc);
    console.log("Getting account details...");
    const accountDetails = await new ChainRestAuthApi(
      network.rest
    ).fetchAccount(walletInjAddr);
    console.log("createWrappedOnInjective...", vaaBytes);
    const msg = await createWrappedOnInjective(tba, walletInjAddr, vaaBytes);
    console.log("cr", msg);

    console.log("submit_vaa", JSON.stringify(msg.params.exec));
    /** Prepare the Transaction **/
    console.log("create transaction...");
    const txFee = DEFAULT_STD_FEE;
    txFee.amount[0] = { amount: "250000000000000", denom: "inj" };
    txFee.gas = "500000";
    const { signBytes, txRaw } = createTransaction({
      message: msg,
      memo: "",
      fee: txFee,
      pubKey: walletPublicKey,
      sequence: parseInt(accountDetails.account.base_account.sequence, 10),
      accountNumber: parseInt(
        accountDetails.account.base_account.account_number,
        10
      ),
      chainId: network.chainId,
    });
    console.log("txRaw", txRaw);

    console.log("sign transaction...");
    /** Sign transaction */
    const signature = await walletPK.sign(Buffer.from(signBytes));

    /** Append Signatures */
    txRaw.signatures = [signature];

    const txService = new TxGrpcApi(network.grpc);

    console.log("simulate transaction...");
    /** Simulate transaction */
    const simulationResponse = await txService.simulate(txRaw);
    console.log(
      `Transaction simulation response: ${JSON.stringify(
        simulationResponse.gasInfo
      )}`
    );

    console.log("broadcast transaction...");
    /** Broadcast transaction */
    const txResponse = await txService.broadcast(txRaw);
    console.log("txResponse", txResponse);

    if (txResponse.code !== 0) {
      console.log(`Transaction failed: ${txResponse.rawLog}`);
    } else {
      console.log(
        `Broadcasted transaction hash: ${JSON.stringify(txResponse.txHash)}`
      );
    }
    const contract = pvaa.Contract || "0";
    console.log("contract", contract);
    const fa = await getForeignAssetInjective(
      tba,
      client,
      "algorand",
      hexToUint8Array(contract)
    );
    console.log("fa", fa);
    const forAsset = fa || "";
    // attested Algo contract = inj10jc4vr9vfq0ykkmfvfgz430w8z6hwdlqhdw9l8
    // Start transfer from Algorand to Injective
    const AmountToTransfer: number = 12300;
    const Fee: number = 0;
    console.log("About to transferFromAlgorand");
    const transferTxs = await transferFromAlgorand(
      algodClient,
      TokenBridgeID,
      CoreID,
      algoWallet.addr,
      AlgoIndex,
      BigInt(AmountToTransfer),
      tryNativeToHexString(walletInjAddr, "injective"),
      CHAIN_ID_INJECTIVE,
      BigInt(Fee)
    );
    console.log("About to signSendAndConfirm");
    const transferResult = await signSendAndConfirmAlgorand(
      algodClient,
      transferTxs,
      algoWallet
    );
    console.log("About to parseSeqFromLog");
    const txSid = parseSequenceFromLogAlgorand(transferResult);
    console.log("About to getSignedVAA");
    const signedVaa = await getSignedVAAWithRetry(
      rpc,
      CHAIN_ID_ALGORAND,
      emitterAddr,
      txSid,
      { transport: NodeHttpTransport() }
    );
    const pv = _parseVAAAlgorand(signedVaa.vaaBytes);
    console.log("vaa", pv);
    console.log("About to redeemOnInjective");
    const roi = await redeemOnInjective(tba, walletInjAddr, signedVaa.vaaBytes);
    console.log("roi", roi);
    {
      const accountDetails = await new ChainRestAuthApi(
        network.rest
      ).fetchAccount(walletInjAddr);
      const { signBytes, txRaw } = createTransaction({
        message: roi,
        memo: "",
        fee: txFee,
        pubKey: walletPublicKey,
        sequence: parseInt(accountDetails.account.base_account.sequence, 10),
        accountNumber: parseInt(
          accountDetails.account.base_account.account_number,
          10
        ),
        chainId: network.chainId,
      });
      console.log("txRaw", txRaw);

      console.log("sign transaction...");
      /** Sign transaction */
      const sig = await walletPK.sign(Buffer.from(signBytes));

      /** Append Signatures */
      txRaw.signatures = [sig];

      const txService = new TxGrpcApi(network.grpc);

      console.log("simulate transaction...");
      /** Simulate transaction */
      const simulationResponse = await txService.simulate(txRaw);
      console.log(
        `Transaction simulation response: ${JSON.stringify(
          simulationResponse.gasInfo
        )}`
      );

      console.log("broadcast transaction...");
      /** Broadcast transaction */
      const txResponse = await txService.broadcast(txRaw);
      console.log("txResponse", txResponse);

      if (txResponse.code !== 0) {
        console.log(`Transaction failed: ${txResponse.rawLog}`);
      } else {
        console.log(
          `Broadcasted transaction hash: ${JSON.stringify(txResponse.txHash)}`
        );
      }
    }
    console.log("Checking if transfer is completed");
    expect(
      await getIsTransferCompletedInjective(tba, signedVaa.vaaBytes, client)
    ).toBe(true);
    {
      console.log("checking vaa:", signedVaa.vaaBytes);
      console.log("checking vaa:", uint8ArrayToHex(signedVaa.vaaBytes));
      const network = getNetworkInfo(getEndPoint());
      const client = new ChainGrpcWasmApi(network.grpc);
      const queryResult = await client.fetchSmartContractState(
        CONTRACTS.TESTNET.injective.token_bridge,
        Buffer.from(
          JSON.stringify({
            transfer_info: {
              vaa: fromUint8Array(signedVaa.vaaBytes),
            },
          })
        ).toString("base64")
      );
      let result: any = null;
      let addr: string = "";
      if (typeof queryResult.data === "string") {
        result = JSON.parse(
          Buffer.from(queryResult.data, "base64").toString("utf-8")
        );
        console.log("result", result);
        addr = tryHexToNativeString(
          uint8ArrayToHex(result.recipient),
          "injective"
        );
        console.log("Injective address?", addr);
      }
      // interface wAlgoBalance {
      //   balance: string;
      // }

      console.log(
        "Getting balance for foreign asset",
        forAsset,
        "on address",
        walletInjAddr
      );
      const balRes = await client.fetchSmartContractState(
        forAsset,
        Buffer.from(
          JSON.stringify({
            balance: {
              address: walletInjAddr,
            },
          })
        ).toString("base64")
      );
      result = null;
      console.log("balRes", balRes);
      if (typeof balRes.data === "string") {
        result = JSON.parse(
          Buffer.from(balRes.data, "base64").toString("utf-8")
        );
        console.log("balRes", result);
      }
    }
  } catch (e) {
    console.error(e);
  }
});

test.skip("Attest and transfer token from Injective to Algorand", async () => {
  const Asset: string = "inj";
  const walletPKHash: string = process.env.ETH_KEY || "";
  const walletPK = PrivateKey.fromPrivateKey(walletPKHash);
  const walletInjAddr = walletPK.toBech32();
  const walletPublicKey = walletPK.toPublicKey().toBase64();

  const network = getNetworkInfo(getEndPoint());
  console.log("create transaction...");
  const txFee = DEFAULT_STD_FEE;
  txFee.amount[0] = { amount: "250000000000000", denom: "inj" };
  txFee.gas = "500000";
  // Attest
  const attestMsg = await attestFromInjective(
    CONTRACTS.TESTNET.injective.token_bridge,
    walletInjAddr,
    Asset
  );

  const accountDetails = await new ChainRestAuthApi(network.rest).fetchAccount(
    walletInjAddr
  );
  const { signBytes, txRaw } = createTransaction({
    message: attestMsg,
    memo: "",
    fee: txFee,
    pubKey: walletPublicKey,
    sequence: parseInt(accountDetails.account.base_account.sequence, 10),
    accountNumber: parseInt(
      accountDetails.account.base_account.account_number,
      10
    ),
    chainId: network.chainId,
  });
  console.log("txRaw", txRaw);

  console.log("sign transaction...");
  /** Sign transaction */
  const signedMsg = await walletPK.sign(Buffer.from(signBytes));

  /** Append Signatures */
  txRaw.signatures = [signedMsg];

  const txService = new TxGrpcApi(network.grpc);

  console.log("simulate transaction...");
  /** Simulate transaction */
  const simulationResponse = await txService.simulate(txRaw);
  console.log(
    `Transaction simulation response: ${JSON.stringify(
      simulationResponse.gasInfo
    )}`
  );

  console.log("broadcast transaction...");
  /** Broadcast transaction */
  const txResponse = await txService.broadcast(txRaw);
  console.log("txResponse", txResponse);

  if (txResponse.code !== 0) {
    console.log(`Transaction failed: ${txResponse.rawLog}`);
  } else {
    console.log(
      `Broadcasted transaction hash: ${JSON.stringify(txResponse.txHash)}`
    );
  }
  console.log("txResponse", JSON.stringify(txResponse.rawLog));
  console.log("txResponse", txResponse.rawLog);
  const sequence = parseSequenceFromLogInjective(txResponse);
  if (!sequence) {
    throw new Error("Sequence not found");
  }
  console.log("found seqNum:", sequence);
  const emitterAddress = await getEmitterAddressInjective(
    CONTRACTS.TESTNET.injective.token_bridge
  );
  const rpc: string[] = ["https://wormhole-v2-testnet-api.certus.one"];
  const { vaaBytes: attestSignedVaa } = await getSignedVAAWithRetry(
    rpc,
    CHAIN_ID_INJECTIVE,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(), //This should only be needed when running in node.
    },
    1000, //retryTimeout
    1000 //Maximum retry attempts
  );
  console.log("signed VAA", uint8ArrayToHex(attestSignedVaa));
  console.log("parsed attestation vaa", _parseVAAAlgorand(attestSignedVaa));

  const algodToken = "";
  const algodServer = "https://testnet-api.algonode.cloud";
  const algodPort = "";
  const algodClient = new Algodv2(algodToken, algodServer, algodPort);

  console.log("Doing Algorand part......");
  console.log("Creating wallet...");
  if (!process.env.ALGO_MNEMONIC) {
    throw new Error("Failed to read in ALGO_MNEMONIC");
  }
  const algoWallet: Account = mnemonicToSecretKey(process.env.ALGO_MNEMONIC);
  const CoreID = BigInt(86525623); // Testnet
  const TokenBridgeID = BigInt(86525641); // Testnet
  console.log("createWrappedOnAlgorand...");
  const createWrappedTxs = await createWrappedOnAlgorand(
    algodClient,
    TokenBridgeID,
    CoreID,
    algoWallet.addr,
    attestSignedVaa
  );
  console.log("signing and sending to algorand...");
  const sscResult = await signSendAndConfirmAlgorand(
    algodClient,
    createWrappedTxs,
    algoWallet
  );
  console.log("sscResult", sscResult);

  console.log("getting foreign asset:");
  let assetIdCreated = await getForeignAssetFromVaaAlgorand(
    algodClient,
    TokenBridgeID,
    attestSignedVaa
  );
  if (!assetIdCreated) {
    throw new Error("Failed to create asset");
  }
  console.log("assetId:", assetIdCreated);
  // Transfer
  const transferMsgs = await transferFromInjective(
    walletInjAddr,
    CONTRACTS.TESTNET.injective.token_bridge,
    "inj",
    "1000000",
    CHAIN_ID_ALGORAND,
    decodeAddress(algoWallet.addr).publicKey
  );
  console.log("number of msgs = ", transferMsgs.length);
  {
    console.log("xferMsgsSigned", transferMsgs);
    const { signBytes, txRaw } = createTransaction({
      message: transferMsgs,
      memo: "",
      fee: txFee,
      pubKey: walletPublicKey,
      sequence:
        parseInt(accountDetails.account.base_account.sequence, 10) +
        transferMsgs.length -
        1,
      accountNumber: parseInt(
        accountDetails.account.base_account.account_number,
        10
      ),
      chainId: network.chainId,
    });
    console.log("txRaw", txRaw);

    console.log("sign transaction...");
    /** Sign transaction */
    const signedMsg = await walletPK.sign(Buffer.from(signBytes));

    /** Append Signatures */
    txRaw.signatures = [signedMsg];

    const txService = new TxGrpcApi(network.grpc);

    console.log("simulate transaction...");
    /** Simulate transaction */
    const simulationResponse = await txService.simulate(txRaw);
    console.log(
      `Transaction simulation response: ${JSON.stringify(
        simulationResponse.gasInfo
      )}`
    );
    console.log("broadcast transaction...");
    /** Broadcast transaction */
    const txResponse = await txService.broadcast(txRaw);
    console.log("txResponse", txResponse);

    if (txResponse.code !== 0) {
      console.log(`Transaction failed: ${txResponse.rawLog}`);
    } else {
      console.log(
        `Broadcasted transaction hash: ${JSON.stringify(txResponse.txHash)}`
      );
    }
    console.log("txResponse", JSON.stringify(txResponse));

    const sequence = parseSequenceFromLogInjective(txResponse);
    if (!sequence) {
      throw new Error("Sequence not found");
    }
    console.log("found seqNum:", sequence);
    const { vaaBytes } = await getSignedVAAWithRetry(
      rpc,
      CHAIN_ID_INJECTIVE,
      emitterAddress,
      sequence,
      {
        transport: NodeHttpTransport(), //This should only be needed when running in node.
      },
      1000, //retryTimeout
      1000 //Maximum retry attempts
    );
    console.log("parsed VAA", _parseVAAAlgorand(vaaBytes));
    console.log("About to redeemOnAlgorand...");
    const tids = await redeemOnAlgorand(
      algodClient,
      TokenBridgeID,
      CoreID,
      vaaBytes,
      algoWallet.addr
    );
    console.log("After redeem...", tids);
    const resToLog = await signSendAndConfirmAlgorand(
      algodClient,
      tids,
      algoWallet
    );
    console.log("resToLog", resToLog["confirmed-round"]);
    console.log("Checking if isRedeemed...");
    const success = await getIsTransferCompletedAlgorand(
      algodClient,
      TokenBridgeID,
      vaaBytes
    );
    expect(success).toBe(true);
  }
  const balances = await getBalances(algodClient, algoWallet.addr);
  console.log("Ending balances", balances);
});
