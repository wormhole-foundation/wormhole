import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  Transaction,
} from "@solana/web3.js";
import { Types } from "aptos";
import { ethers, Overrides } from "ethers";
import { Bridge__factory } from "../ethers-contracts";
import {
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
  createCompleteWrappedMetaInstruction,
} from "../solana/nftBridge";
import { CHAIN_ID_APTOS, CHAIN_ID_SOLANA } from "../utils";
import { parseNftTransferVaa, parseVaa, SignedVaa } from "../vaa";

export async function redeemOnEth(
  nftBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array,
  overrides: Overrides & { from?: string | Promise<string> } = {}
): Promise<ethers.ContractReceipt> {
  const bridge = Bridge__factory.connect(nftBridgeAddress, signer);
  const v = await bridge.completeTransfer(signedVAA, overrides);
  const receipt = await v.wait();
  return receipt;
}

export async function isNFTVAASolanaNative(
  signedVAA: Uint8Array
): Promise<boolean> {
  return parseVaa(signedVAA).payload.readUInt16BE(33) === CHAIN_ID_SOLANA;
}

export async function redeemOnSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  nftBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  signedVaa: SignedVaa,
  toAuthorityAddress?: PublicKeyInitData,
  commitment?: Commitment
): Promise<Transaction> {
  const parsed = parseNftTransferVaa(signedVaa);
  const createCompleteTransferInstruction =
    parsed.tokenChain == CHAIN_ID_SOLANA
      ? createCompleteTransferNativeInstruction
      : createCompleteTransferWrappedInstruction;
  const transaction = new Transaction().add(
    createCompleteTransferInstruction(
      nftBridgeAddress,
      bridgeAddress,
      payerAddress,
      parsed,
      toAuthorityAddress
    )
  );
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}

export async function createMetaOnSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  nftBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  signedVaa: SignedVaa,
  commitment?: Commitment
): Promise<Transaction> {
  const parsed = parseNftTransferVaa(signedVaa);
  if (parsed.tokenChain == CHAIN_ID_SOLANA) {
    return Promise.reject("parsed.tokenChain == CHAIN_ID_SOLANA");
  }
  const transaction = new Transaction().add(
    createCompleteWrappedMetaInstruction(
      nftBridgeAddress,
      bridgeAddress,
      payerAddress,
      parsed
    )
  );
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}

export async function redeemOnAptos(
  nftBridgeAddress: string,
  transferVAA: Uint8Array
): Promise<Types.EntryFunctionPayload> {
  const parsedVAA = parseNftTransferVaa(transferVAA);
  if (parsedVAA.toChain !== CHAIN_ID_APTOS) {
    throw new Error("Transfer is not destined for Aptos.");
  }

  return {
    function: `${nftBridgeAddress}::complete_transfer::submit_vaa_and_register_entry`,
    type_arguments: [],
    arguments: [transferVAA],
  };
}
