import * as anchor from "@coral-xyz/anchor";
import { bs58 } from "@coral-xyz/anchor/dist/cjs/utils/bytes";
import { PublicKey } from "@solana/web3.js";
import { expect } from "chai";
import WormholePostMessageShimIdl from "../idls/wormhole_post_message_shim.json";

export async function getSequenceFromTx(
  tx: string
): Promise<{ emitter: PublicKey; sequence: bigint }> {
  const txDetails = await getTransactionDetails(tx);

  const borshEventCoder = new anchor.BorshEventCoder(
    WormholePostMessageShimIdl as any
  );

  const innerInstructions = txDetails.meta.innerInstructions[0].instructions;

  // Get the last instruction from the inner instructions
  const lastInstruction = innerInstructions[innerInstructions.length - 1];

  // Decode the Base58 encoded data
  const decodedData = bs58.decode(lastInstruction.data);

  // Remove the instruction discriminator and re-encode the rest as Base58
  const eventData = Buffer.from(decodedData.subarray(8)).toString("base64");

  const borshEvents = borshEventCoder.decode(eventData);
  expect(txDetails.blockTime).is.not.undefined;
  expect(borshEvents.data.submission_time).to.equal(txDetails.blockTime);

  return {
    emitter: borshEvents.data.emitter,
    sequence: BigInt(borshEvents.data.sequence.toString()),
  };
}

export async function getTransactionDetails(
  tx: string
): Promise<anchor.web3.VersionedTransactionResponse> {
  let txDetails: anchor.web3.VersionedTransactionResponse | null = null;
  while (!txDetails) {
    txDetails = await anchor.getProvider().connection.getTransaction(tx, {
      maxSupportedTransactionVersion: 0,
      commitment: "confirmed",
    });
  }
  return txDetails;
}

export async function logCostAndCompute(method: string, tx: string) {
  const SOL_PRICE = 217.54; // 2025-01-03
  const txDetails = await getTransactionDetails(tx);
  const lamports =
    txDetails.meta.preBalances[0] - txDetails.meta.postBalances[0];
  const sol = lamports / 1_000_000_000;
  console.log(
    `${method}: lamports ${lamports} SOL ${sol}, $${sol * SOL_PRICE}, CU ${
      txDetails.meta.computeUnitsConsumed
    }, tx https://explorer.solana.com/tx/${tx}?cluster=custom&customUrl=http%3A%2F%2Flocalhost%3A8899`
  );
}
