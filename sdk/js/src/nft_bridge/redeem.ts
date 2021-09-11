import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { ethers } from "ethers";
import { CHAIN_ID_SOLANA } from "..";
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
  signedVAA: Uint8Array
) {
  const { parse_vaa } = await import("../solana/core/bridge");
  const parsedVAA = parse_vaa(signedVAA);
  const isSolanaNative =
    Buffer.from(new Uint8Array(parsedVAA.payload)).readUInt16BE(33) ===
    CHAIN_ID_SOLANA;
  const { complete_transfer_wrapped_ix, complete_transfer_native_ix } =
    await import("../solana/nft/nft_bridge");
  const ixs = [];
  if (isSolanaNative) {
    ixs.push(
      ixFromRust(
        complete_transfer_native_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          payerAddress, //TODO: allow for a different address than payer
          signedVAA
        )
      )
    );
  } else {
    ixs.push(
      ixFromRust(
        complete_transfer_wrapped_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          payerAddress, //TODO: allow for a different address than payer
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
