import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
} from "./consts";
import Wallet from "@project-serum/sol-wallet-adapter";
import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { postVaa } from "./postVaa";
import { ixFromRust } from "../sdk";

export async function redeemOnEth(
  provider: ethers.providers.Web3Provider | undefined,
  signer: ethers.Signer | undefined,
  signedVAA: Uint8Array
) {
  console.log(provider, signer, signedVAA);
  if (!provider || !signer) return;
  console.log("completing transfer");
  const bridge = Bridge__factory.connect(ETH_TOKEN_BRIDGE_ADDRESS, signer);
  const v = await bridge.completeTransfer(signedVAA);
  const receipt = await v.wait();
  console.log(receipt);
}

export async function redeemOnSolana(
  wallet: Wallet | undefined,
  payerAddress: string | undefined, //TODO: we may not need this since we have wallet
  signedVAA: Uint8Array
) {
  if (!wallet || !wallet.publicKey || !payerAddress) return;
  console.log("completing transfer");
  console.log("PROGRAM:", SOL_TOKEN_BRIDGE_ADDRESS);
  console.log("BRIDGE:", SOL_BRIDGE_ADDRESS);
  console.log("PAYER:", payerAddress);
  console.log("VAA:", signedVAA);
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const { complete_transfer_wrapped_ix } = await import("token-bridge");

  await postVaa(
    connection,
    wallet,
    SOL_BRIDGE_ADDRESS,
    payerAddress,
    Buffer.from(signedVAA)
  );
  console.log(Buffer.from(signedVAA).toString("hex"));
  const ix = ixFromRust(
    complete_transfer_wrapped_ix(
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

const redeemOn = {
  [CHAIN_ID_ETH]: redeemOnEth,
  [CHAIN_ID_SOLANA]: redeemOnSolana,
};

export default redeemOn;
