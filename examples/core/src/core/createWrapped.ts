import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  createWrappedOnEth,
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

//This function is quite similar to redeem, but invokes createWrapped, which 'attests' an asset on a foreign
//chain by submitting its VAA. This creates the Wormhole-Wrapped asset on the target chain, hence the name.
export async function createWrapped(
  targetChain: ChainId,
  signedVaa: Uint8Array
) {
  if (
    targetChain === CHAIN_ID_ETH ||
    targetChain === CHAIN_ID_POLYGON ||
    targetChain === CHAIN_ID_BSC
  ) {
    await createWrappedEvm(signedVaa, targetChain);
  } else if (targetChain === CHAIN_ID_SOLANA) {
    await createWrappedSolana(signedVaa);
  } else if (targetChain === CHAIN_ID_TERRA) {
    await createWrappedTerra(signedVaa);
  } else {
    return;
  }
}

export async function createWrappedEvm(
  signedVAA: Uint8Array,
  targetChain: ChainId
) {
  const signer: Signer = getSignerForChain(targetChain);
  try {
    return await createWrappedOnEth(
      getTokenBridgeAddressForChain(targetChain),
      signer,
      signedVAA
    );
  } catch (e) {
    //console.error(e);
    //createWrapped throws an exception if the wrapped asset is already created.
  }
}

export async function createWrappedSolana(signedVAA: Uint8Array) {
  //TODO this
  return Promise.resolve();
}

export async function createWrappedTerra(signedVAA: Uint8Array) {
  //TODO adapt bridge_ui implementation to use in-memory terra key
}
