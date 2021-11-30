import { hexToUint8Array, postVaaSolana } from "@certusone/wormhole-sdk";
import { redeemOnSolana } from "@certusone/wormhole-sdk";
import { Connection, Keypair } from "@solana/web3.js";
import { TextEncoder } from "util";
import { ChainConfigInfo } from "../configureEnv";

export async function relaySolana(
  chainConfigInfo: ChainConfigInfo,
  signedVAA: string,
  unwrapNative: boolean,
  request: any,
  response: any
) {
  //TODO native transfer & create associated token account
  //TODO close connection
  const connection = new Connection(chainConfigInfo.nodeUrl, "confirmed");
  const keypair = Keypair.fromSecretKey(
    new TextEncoder().encode(chainConfigInfo.walletPrivateKey)
  );
  const payerAddress = keypair.publicKey.toString();
  await postVaaSolana(
    connection,
    async (transaction) => {
      transaction.partialSign(keypair);
      return transaction;
    },
    chainConfigInfo.bridgeAddress,
    payerAddress,
    Buffer.from(signedVAA)
  );
  const receipt = await redeemOnSolana(
    connection,
    chainConfigInfo.bridgeAddress,
    chainConfigInfo.tokenBridgeAddress,
    payerAddress,
    hexToUint8Array(signedVAA)
  );
  response.status(200).json(receipt);
}
