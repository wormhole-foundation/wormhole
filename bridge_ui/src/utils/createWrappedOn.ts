import Wallet from "@project-serum/sol-wallet-adapter";
import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { ixFromRust } from "../sdk";
import {
  CHAIN_ID_SOLANA,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  ETH_TOKEN_BRIDGE_ADDRESS,
  CHAIN_ID_ETH,
} from "./consts";
import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { postVaa } from "./postVaa";

export async function createWrappedOnEth(
  provider: ethers.providers.Web3Provider | undefined,
  signer: ethers.Signer | undefined,
  signedVAA: Uint8Array
) {
  console.log(provider, signer, signedVAA);
  if (!provider || !signer) return;
  console.log("creating wrapped");
  const bridge = Bridge__factory.connect(ETH_TOKEN_BRIDGE_ADDRESS, signer);
  const v = await bridge.createWrapped(signedVAA);
  const receipt = await v.wait();
  console.log(receipt);
}

export async function createWrappedOnSolana(
  wallet: Wallet | undefined,
  payerAddress: string | undefined, //TODO: we may not need this since we have wallet
  signedVAA: Uint8Array
) {
  if (!wallet || !wallet.publicKey || !payerAddress) return;
  console.log("creating wrapped");
  console.log("PROGRAM:", SOL_TOKEN_BRIDGE_ADDRESS);
  console.log("BRIDGE:", SOL_BRIDGE_ADDRESS);
  console.log("PAYER:", payerAddress);
  console.log("VAA:", signedVAA);
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const { create_wrapped_ix } = await import("token-bridge");

  await postVaa(
    connection,
    wallet,
    SOL_BRIDGE_ADDRESS,
    payerAddress,
    Buffer.from(signedVAA)
  );
  const ix = ixFromRust(
    create_wrapped_ix(
      SOL_TOKEN_BRIDGE_ADDRESS,
      SOL_BRIDGE_ADDRESS,
      payerAddress,
      signedVAA
    )
  );
  const transaction = new Transaction().add(ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  // Sign transaction, broadcast, and confirm
  const signed = await wallet.signTransaction(transaction);
  console.log("SIGNED", signed);
  const txid = await connection.sendRawTransaction(signed.serialize());
  console.log("SENT", txid);
  const conf = await connection.confirmTransaction(txid);
  console.log("CONFIRMED", conf);
  const info = await connection.getTransaction(txid);
  console.log("INFO", info);
}

const createWrappedOn = {
  [CHAIN_ID_SOLANA]: createWrappedOnSolana,
  [CHAIN_ID_ETH]: createWrappedOnEth,
};

export default createWrappedOn;
