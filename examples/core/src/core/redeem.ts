import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  postVaaSolana,
  redeemOnEth,
  redeemOnSolana,
} from "@certusone/wormhole-sdk";
import { Signer } from "@ethersproject/abstract-signer";
import { Connection, Keypair } from "@solana/web3.js";
import {
  getNFTBridgeAddressForChain,
  getSignerForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "../consts";

//This function attempts to redeem the VAA on the target chain.
export async function redeem(
  targetChain: ChainId,
  signedVaa: Uint8Array,
  isNftTransfer: boolean
) {
  if (
    targetChain === CHAIN_ID_ETH ||
    targetChain === CHAIN_ID_POLYGON ||
    targetChain === CHAIN_ID_BSC
  ) {
    await redeemEvm(signedVaa, targetChain, isNftTransfer);
  } else if (targetChain === CHAIN_ID_SOLANA) {
    await redeemSolana(signedVaa, isNftTransfer);
  } else if (targetChain === CHAIN_ID_TERRA) {
    await redeemTerra(signedVaa);
  } else {
    return;
  }
}

export async function redeemEvm(
  signedVAA: Uint8Array,
  targetChain: ChainId,
  isNftTransfer: boolean
) {
  const signer: Signer = getSignerForChain(targetChain);
  try {
    await redeemOnEth(
      isNftTransfer
        ? getNFTBridgeAddressForChain(targetChain)
        : getTokenBridgeAddressForChain(targetChain),
      signer,
      signedVAA
    );
  } catch (e) {}
}

export async function redeemSolana(
  signedVAA: Uint8Array,
  isNftTransfer: boolean
) {
  if (isNftTransfer) {
    //TODO
    //Solana redemptions require sending metadata to the chain inside of transactions,
    //and this in not yet available in the sdk.
    return;
  }
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
  const payerAddress = keypair.publicKey.toString();
  await postVaaSolana(
    connection,
    async (transaction) => {
      transaction.partialSign(keypair);
      return transaction;
    },
    SOL_BRIDGE_ADDRESS,
    payerAddress,
    Buffer.from(signedVAA)
  );
  await redeemOnSolana(
    connection,
    SOL_BRIDGE_ADDRESS,
    SOL_TOKEN_BRIDGE_ADDRESS,
    payerAddress,
    signedVAA
  );
}

export async function redeemTerra(signedVAA: Uint8Array) {
  //TODO adapt bridge_ui implementation to use in-memory terra key
}
