import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  transferFromEth as transferFromEthTx,
  transferFromSolana as transferFromSolanaTx,
} from "@certusone/wormhole-sdk";
import Wallet from "@project-serum/sol-wallet-adapter";
import { Connection } from "@solana/web3.js";
import { ethers } from "ethers";
import { arrayify, parseUnits, zeroPad } from "ethers/lib/utils";
import { hexToUint8Array } from "./array";
import {
  ETH_BRIDGE_ADDRESS,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "./consts";
import { getSignedVAAWithRetry } from "./getSignedVAAWithRetry";
import { signSendConfirmAndGet } from "./solana";

// TODO: overall better input checking and error handling
export async function transferFromEth(
  signer: ethers.Signer | undefined,
  tokenAddress: string,
  decimals: number,
  amount: string,
  recipientChain: ChainId,
  recipientAddress: Uint8Array | undefined
) {
  if (!signer || !recipientAddress) return;
  //TODO: check if token attestation exists on the target chain
  const amountParsed = parseUnits(amount, decimals);
  const receipt = await transferFromEthTx(
    ETH_TOKEN_BRIDGE_ADDRESS,
    signer,
    tokenAddress,
    amountParsed,
    recipientChain,
    recipientAddress
  );
  const sequence = parseSequenceFromLogEth(receipt, ETH_BRIDGE_ADDRESS);
  const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
  const { vaaBytes } = await getSignedVAAWithRetry(
    CHAIN_ID_ETH,
    emitterAddress,
    sequence.toString()
  );
  return vaaBytes;
}

export async function transferFromSolana(
  wallet: Wallet | undefined,
  payerAddress: string | undefined, //TODO: we may not need this since we have wallet
  fromAddress: string | undefined,
  mintAddress: string,
  amount: string,
  decimals: number,
  targetAddressStr: string | undefined,
  targetChain: ChainId,
  originAddressStr?: string,
  originChain?: ChainId
) {
  if (
    !wallet ||
    !wallet.publicKey ||
    !payerAddress ||
    !fromAddress ||
    !targetAddressStr ||
    (originChain && !originAddressStr)
  )
    return;
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const targetAddress = zeroPad(arrayify(targetAddressStr), 32);
  const amountParsed = parseUnits(amount, decimals).toBigInt();
  const originAddress = originAddressStr
    ? zeroPad(hexToUint8Array(originAddressStr), 32)
    : undefined;
  const transaction = await transferFromSolanaTx(
    connection,
    SOL_BRIDGE_ADDRESS,
    SOL_TOKEN_BRIDGE_ADDRESS,
    payerAddress,
    fromAddress,
    mintAddress,
    amountParsed,
    targetAddress,
    targetChain,
    originAddress,
    originChain
  );
  const info = await signSendConfirmAndGet(wallet, connection, transaction);
  if (!info) {
    throw new Error("An error occurred while fetching the transaction info");
  }
  const sequence = parseSequenceFromLogSolana(info);
  const emitterAddress = await getEmitterAddressSolana(
    SOL_TOKEN_BRIDGE_ADDRESS
  );
  const { vaaBytes } = await getSignedVAAWithRetry(
    CHAIN_ID_SOLANA,
    emitterAddress,
    sequence
  );
  return vaaBytes;
}
