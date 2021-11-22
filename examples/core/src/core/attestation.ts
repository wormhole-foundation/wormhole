import {
  attestFromEth,
  attestFromSolana,
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
} from "@certusone/wormhole-sdk";
import { Connection, Keypair } from "@solana/web3.js";
import {
  getBridgeAddressForChain,
  getSignerForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "../consts";

/*
This function attests the given token and returns the resultant sequence number, which is then used to retrieve the
VAA from the guardians.
*/
export async function attest(
  originChain: ChainId,
  originAsset: string
): Promise<string> {
  if (originChain === CHAIN_ID_SOLANA) {
    return attestSolana(originAsset);
  } else if (originChain === CHAIN_ID_TERRA) {
    return attestTerra(originAsset);
  } else {
    return attestEvm(originChain, originAsset);
  }
}

export async function attestEvm(
  originChain: ChainId,
  originAsset: string
): Promise<string> {
  const signer = getSignerForChain(originChain);
  const receipt = await attestFromEth(
    getTokenBridgeAddressForChain(originChain),
    signer,
    originAsset
  );
  const sequence = parseSequenceFromLogEth(
    receipt,
    getBridgeAddressForChain(originChain)
  );
  return sequence;
}

export async function attestTerra(originAsset: string): Promise<string> {
  //TODO modify bridge_ui to use in-memory signer
  throw new Error("Unimplemented");
}

export async function attestSolana(originAsset: string): Promise<string> {
  const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
  const payerAddress = keypair.publicKey.toString();
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const transaction = await attestFromSolana(
    connection,
    SOL_BRIDGE_ADDRESS,
    SOL_TOKEN_BRIDGE_ADDRESS,
    payerAddress,
    originAsset
  );
  transaction.partialSign(keypair);
  const txid = await connection.sendRawTransaction(transaction.serialize());
  await connection.confirmTransaction(txid);
  const info = await connection.getTransaction(txid);
  if (!info) {
    throw new Error("An error occurred while fetching the transaction info");
  }
  const sequence = parseSequenceFromLogSolana(info);
  return sequence;
}
