import {
  postVaaSolana,
  redeemOnEth as redeemOnEthTx,
  redeemOnSolana as redeemOnSolanaTx,
} from "@certusone/wormhole-sdk";
import Wallet from "@project-serum/sol-wallet-adapter";
import { Connection } from "@solana/web3.js";
import { ethers } from "ethers";
import {
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "./consts";
import { signSendAndConfirm } from "./solana";

export async function redeemOnEth(
  signer: ethers.Signer | undefined,
  signedVAA: Uint8Array
) {
  if (!signer) return;
  await redeemOnEthTx(ETH_TOKEN_BRIDGE_ADDRESS, signer, signedVAA);
}

export async function redeemOnSolana(
  wallet: Wallet | undefined,
  payerAddress: string | undefined, //TODO: we may not need this since we have wallet
  signedVAA: Uint8Array,
  isSolanaNative: boolean,
  mintAddress?: string // TODO: read the signedVAA and create the account if it doesn't exist
) {
  if (!wallet || !wallet.publicKey || !payerAddress) return;
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  await postVaaSolana(
    connection,
    wallet,
    SOL_BRIDGE_ADDRESS,
    payerAddress,
    Buffer.from(signedVAA)
  );
  const transaction = await redeemOnSolanaTx(
    connection,
    SOL_BRIDGE_ADDRESS,
    SOL_TOKEN_BRIDGE_ADDRESS,
    payerAddress,
    signedVAA,
    isSolanaNative,
    mintAddress
  );
  await signSendAndConfirm(wallet, connection, transaction);
}
