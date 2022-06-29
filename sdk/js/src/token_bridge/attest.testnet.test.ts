import { getNetworkInfo, Network } from "@injectivelabs/networks";
import {
  ChainRestAuthApi,
  DEFAULT_STD_FEE,
  MsgExecuteContract,
  privateKeyToPublicKeyBase64,
} from "@injectivelabs/sdk-ts";
import {
  createTransaction,
  TxClient,
  TxGrpcClient,
} from "@injectivelabs/tx-ts";
import { PrivateKey } from "@injectivelabs/sdk-ts/dist/local";
import { test } from "@jest/globals";
import { CONTRACTS } from "..";

test.skip("testnet - injective attest native token", async () => {
  const network = getNetworkInfo(Network.TestnetK8s);
  console.log("Using network:", network);
  const privateKeyHash = process.env.ETH_KEY || "";
  const privateKey = PrivateKey.fromPrivateKey(privateKeyHash);
  const injectiveAddress = privateKey.toBech32();
  console.log("Using wallet:", injectiveAddress);
  const publicKey = privateKeyToPublicKeyBase64(
    Buffer.from(privateKeyHash, "hex")
  );
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
  const msg = MsgExecuteContract.fromJSON({
    contractAddress: CONTRACTS.TESTNET.injective.token_bridge,
    sender: injectiveAddress,
    msg: {
      create_asset_meta: {
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
    },
  });

  /** Prepare the Transaction **/
  console.log("Prepare the transaction");
  const { signBytes, txRaw } = createTransaction({
    message: msg.toDirectSign(),
    memo: "",
    fee: DEFAULT_STD_FEE,
    pubKey: Buffer.from(publicKey).toString("base64"),
    sequence: parseInt(accountDetails.account.base_account.sequence, 10),
    accountNumber: parseInt(
      accountDetails.account.base_account.account_number,
      10
    ),
    chainId: network.chainId,
  });

  /** Sign transaction */
  console.log("Sign transaction");
  const signature = await privateKey.sign(signBytes);

  /** Append Signatures */
  console.log("Append signatures");
  txRaw.setSignaturesList([signature]);

  /** Calculate hash of the transaction */
  console.log("Calculate hash");
  console.log(`Transaction Hash: ${await TxClient.hash(txRaw)}`);

  const txService = new TxGrpcClient({
    txRaw,
    endpoint: network.sentryGrpcApi,
  });

  /** Simulate transaction */
  console.log("Simulate transaction");
  const simulationResponse = await txService.simulate();
  console.log(
    `Transaction simulation response: ${JSON.stringify(
      simulationResponse.gasInfo
    )}`
  );

  /** Broadcast transaction */
  console.log("Broadcast transaction");
  const txResponse = await txService.broadcast();
  console.log(
    `Broadcasted transaction hash: ${JSON.stringify(txResponse.txhash)}`
  );

  console.log(txResponse);
});
