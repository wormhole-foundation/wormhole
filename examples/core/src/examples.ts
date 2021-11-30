import {
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  redeemOnEth,
  WSOL_ADDRESS,
} from "@certusone/wormhole-sdk";
import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import { relay } from "./basicRelayer";
import { fullAttestation } from "./commonWorkflows";
import {
  ETH_TEST_WALLET_PUBLIC_KEY,
  getSignerForChain,
  getTokenBridgeAddressForChain,
  SOLANA_TEST_TOKEN,
  SOLANA_TEST_WALLET_PUBLIC_KEY,
  WETH_ADDRESS,
} from "./consts";
/*
The goal of this example program is to demonstrate a common Wormhole token bridge
use-case.

*/
import { attest } from "./core/attestation";
import { createWrapped } from "./core/createWrapped";
import { getSignedVAABySequence } from "./core/guardianQuery";
import { redeem } from "./core/redeem";
import { transferTokens } from "./core/transfer";
setDefaultWasm("node");

/*
This example attests a test token on Solana, retrieves the resulting VAA, and then submits it
to Ethereum, thereby registering the token on Ethereum.
*/
export async function attestWETH() {
  return fullAttestation(CHAIN_ID_ETH, WETH_ADDRESS);
}

export async function attestWBNB() {
  return fullAttestation(CHAIN_ID_BSC, WETH_ADDRESS);
}

export async function bridgeWsolToEthereum() {
  const sequenceNumber = await transferTokens(
    CHAIN_ID_SOLANA,
    "20.0",
    CHAIN_ID_ETH,
    SOLANA_TEST_WALLET_PUBLIC_KEY, //When transferring native SOL, use the native / payer address.
    ETH_TEST_WALLET_PUBLIC_KEY,
    true
  );

  const signedVaa = await getSignedVAABySequence(
    CHAIN_ID_SOLANA,
    sequenceNumber,
    false
  );

  await redeemOnEth(
    getTokenBridgeAddressForChain(CHAIN_ID_ETH),
    getSignerForChain(CHAIN_ID_ETH),
    signedVaa
  );
}

export async function transferWithRelayHandoff() {
  const sourceAddress = (
    await Token.getAssociatedTokenAddress(
      ASSOCIATED_TOKEN_PROGRAM_ID,
      TOKEN_PROGRAM_ID,
      new PublicKey(SOLANA_TEST_TOKEN),
      new PublicKey(SOLANA_TEST_WALLET_PUBLIC_KEY)
    )
  ).toString();

  const sequenceNumber = await transferTokens(
    CHAIN_ID_SOLANA,
    "1.0",
    CHAIN_ID_ETH,
    sourceAddress,
    ETH_TEST_WALLET_PUBLIC_KEY,
    false,
    SOLANA_TEST_TOKEN,
    9
  );

  await relay(CHAIN_ID_SOLANA, sequenceNumber, false);
}
