import { getNetworkInfo, Network } from "@injectivelabs/networks";
import { DEFAULT_STD_FEE, getStdFee } from "@injectivelabs/utils";
import {
  TxClient,
  PrivateKey,
  TxGrpcApi,
  ChainRestAuthApi,
  createTransaction,
  MsgExecuteContractCompat,
} from "@injectivelabs/sdk-ts";
import { test } from "@jest/globals";
import { CONTRACTS } from "..";

test.skip("testnet - injective attest native token", async () => {
  const network = getNetworkInfo(Network.TestnetK8s);
  console.log("Using network:", network);
  const privateKeyHash = process.env.ETH_KEY || "";
  const privateKey = PrivateKey.fromHex(privateKeyHash);
  const injectiveAddress = privateKey.toBech32();
  console.log("Using wallet:", injectiveAddress);
  const publicKey = privateKey.toPublicKey().toBase64();
  const isNativeAsset = true;
  const asset = "inj";
  const nonce = 69;

  console.log("Account details");

  /** Account Details **/
  const accountDetails = await new ChainRestAuthApi(
    network.sentryHttpApi
  ).fetchAccount(injectiveAddress);
  console.log(accountDetails);

  /** Prepare the Message */
  console.log("Prepare the message");
  const msg = MsgExecuteContractCompat.fromJSON({
    contractAddress: CONTRACTS.TESTNET.injective.token_bridge,
    sender: injectiveAddress,
    exec: {
      msg: {
        asset_info: isNativeAsset
          ? {
              native_token: { denom: asset },
            }
          : {
              token: {
                contract_addr: asset,
              },
            },
        nonce: nonce,
      },
      action: "create_asset_meta",
    },
  });

  /** Prepare the Transaction **/
  console.log("Prepare the transaction");
  const { signBytes, txRaw } = createTransaction({
    message: msg,
    memo: "",
    fee: getStdFee((parseInt(DEFAULT_STD_FEE.gas, 10) * 2.5).toString()),
    pubKey: publicKey,
    sequence: parseInt(accountDetails.account.base_account.sequence, 10),
    accountNumber: parseInt(
      accountDetails.account.base_account.account_number,
      10
    ),
    chainId: network.chainId,
  });

  /** Sign transaction */
  console.log("Sign transaction");
  const signature = await privateKey.sign(Buffer.from(signBytes));

  /** Append Signatures */
  console.log("Append signatures");
  txRaw.signatures = [signature];

  /** Calculate hash of the transaction */
  console.log("Calculate hash");
  console.log(`Transaction Hash: ${await TxClient.hash(txRaw)}`);

  const txService = new TxGrpcApi(network.sentryGrpcApi);

  /** Simulate transaction */
  console.log("Simulate transaction");
  const simulationResponse = await txService.simulate(txRaw);
  console.log(
    `Transaction simulation response: ${JSON.stringify(
      simulationResponse.gasInfo
    )}`
  );

  /** Broadcast transaction */
  console.log("Broadcast transaction");
  const txResponse = await txService.broadcast(txRaw);
  console.log(
    `Broadcasted transaction hash: ${JSON.stringify(txResponse.txHash)}`
  );

  console.log(txResponse);
});
