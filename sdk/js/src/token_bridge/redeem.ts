import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { ethers } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import { ixFromRust } from "../solana";

export async function redeemOnEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array
) {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.completeTransfer(signedVAA);
  const receipt = await v.wait();
  return receipt;
}

export async function redeemOnSolana(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  signedVAA: Uint8Array,
  isSolanaNative: boolean,
  mintAddress?: string // TODO: read the signedVAA and create the account if it doesn't exist
) {
  const { complete_transfer_wrapped_ix, complete_transfer_native_ix } =
    await import("../solana/token/token_bridge");
  const ixs = [];
  if (isSolanaNative) {
    console.log("COMPLETE TRANSFER NATIVE");
    ixs.push(
      ixFromRust(
        complete_transfer_native_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          signedVAA
        )
      )
    );
  } else {
    // TODO: we should always do this, they could buy wrapped somewhere else and transfer it back for the first time, but again, do it based on vaa
    if (mintAddress) {
      const mintPublicKey = new PublicKey(mintAddress);
      // TODO: re: todo above, this should be swapped for the address from the vaa (may not be the same as the payer)
      const payerPublicKey = new PublicKey(payerAddress);
      const associatedAddress = await Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        mintPublicKey,
        payerPublicKey
      );
      const associatedAddressInfo = await connection.getAccountInfo(
        associatedAddress
      );
      if (!associatedAddressInfo) {
        ixs.push(
          await Token.createAssociatedTokenAccountInstruction(
            ASSOCIATED_TOKEN_PROGRAM_ID,
            TOKEN_PROGRAM_ID,
            mintPublicKey,
            associatedAddress,
            payerPublicKey, // owner
            payerPublicKey // payer
          )
        );
      }
    }
    ixs.push(
      ixFromRust(
        complete_transfer_wrapped_ix(
          tokenBridgeAddress,
          bridgeAddress,
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
  return transaction;
}
