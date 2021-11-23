import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { setDefaultWasm } from "@certusone/wormhole-sdk/cjs/esm/solana/wasm";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";
import { relay } from "./basicRelayer";
import {
  ETH_TEST_WALLET_PUBLIC_KEY,
  SOLANA_TEST_TOKEN,
  SOLANA_TEST_WALLET_PUBLIC_KEY,
} from "./consts";
/*
The goal of this example program is to demonstrate a common Wormhole token bridge
use-case.

*/
import { attest } from "./core/attestation";
import { getSignedVAABySequence } from "./core/guardianQuery";
import { redeem } from "./core/redeem";
import { transferTokens } from "./core/transfer";
setDefaultWasm("node");

/*
This example attests a test token on Solana, retrieves the resulting VAA, and then submits it
to Ethereum, thereby registering the token on Ethereum.
*/
export async function attestationExample() {
  const sequenceNumber = await attest(CHAIN_ID_SOLANA, SOLANA_TEST_TOKEN);

  const signedVaa = await getSignedVAABySequence(
    CHAIN_ID_SOLANA,
    sequenceNumber,
    false
  );
  await redeem(CHAIN_ID_ETH, signedVaa, false);
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
