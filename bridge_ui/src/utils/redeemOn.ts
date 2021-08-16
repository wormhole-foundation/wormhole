import {
  Bridge__factory,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  ixFromRust,
} from "@certusone/wormhole-sdk";
import Wallet from "@project-serum/sol-wallet-adapter";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { ethers } from "ethers";
import {
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "./consts";
import { postVaa } from "./postVaa";

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
  signedVAA: Uint8Array,
  isSolanaNative: boolean,
  mintAddress?: string // TODO: read the signedVAA and create the account if it doesn't exist
) {
  if (!wallet || !wallet.publicKey || !payerAddress) return;
  console.log("completing transfer");
  console.log("PROGRAM:", SOL_TOKEN_BRIDGE_ADDRESS);
  console.log("BRIDGE:", SOL_BRIDGE_ADDRESS);
  console.log("PAYER:", payerAddress);
  console.log("VAA:", signedVAA);
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const { complete_transfer_wrapped_ix, complete_transfer_native_ix } =
    await import("@certusone/wormhole-sdk/lib/solana/token/token_bridge");

  await postVaa(
    connection,
    wallet,
    SOL_BRIDGE_ADDRESS,
    payerAddress,
    Buffer.from(signedVAA)
  );
  console.log(Buffer.from(signedVAA).toString("hex"));
  const ixs = [];
  if (isSolanaNative) {
    console.log("COMPLETE TRANSFER NATIVE");
    ixs.push(
      ixFromRust(
        complete_transfer_native_ix(
          SOL_TOKEN_BRIDGE_ADDRESS,
          SOL_BRIDGE_ADDRESS,
          payerAddress,
          signedVAA
        )
      )
    );
  } else {
    // TODO: we should always do this, they could buy wrapped somewhere else and transfer it back for the first time, but again, do it based on vaa
    if (mintAddress) {
      console.log("CHECK ASSOCIATED TOKEN ACCOUNT FOR", mintAddress);
      const mintPublicKey = new PublicKey(mintAddress);
      const associatedAddress = await Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        mintPublicKey,
        wallet.publicKey
      );
      const associatedAddressInfo = await connection.getAccountInfo(
        associatedAddress
      );
      console.log(
        "CREATE ASSOCIATED TOKEN ACCOUNT",
        associatedAddress.toString()
      );
      if (!associatedAddressInfo) {
        ixs.push(
          await Token.createAssociatedTokenAccountInstruction(
            ASSOCIATED_TOKEN_PROGRAM_ID,
            TOKEN_PROGRAM_ID,
            mintPublicKey,
            associatedAddress,
            wallet.publicKey,
            wallet.publicKey
          )
        );
      }
    }
    console.log("COMPLETE TRANSFER WRAPPED");
    ixs.push(
      ixFromRust(
        complete_transfer_wrapped_ix(
          SOL_TOKEN_BRIDGE_ADDRESS,
          SOL_BRIDGE_ADDRESS,
          payerAddress,
          signedVAA
        )
      )
    );
  }
  const transaction = new Transaction().add(...ixs);
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
