import { parseUnits } from "@ethersproject/units";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { ethers } from "ethers";
import {
  approveEth,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  getForeignAssetSolana,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  transferFromEth,
} from "../..";
import {
  ETH_CORE_BRIDGE_ADDRESS,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOLANA_TOKEN_BRIDGE_ADDRESS,
  TEST_ERC20,
} from "./consts";

export async function transferFromEthToSolana(): Promise<string> {
  // create a keypair for Solana
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
  // determine destination address - an associated token account
  const solanaMintKey = new PublicKey(
    (await getForeignAssetSolana(
      connection,
      SOLANA_TOKEN_BRIDGE_ADDRESS,
      CHAIN_ID_ETH,
      hexToUint8Array(nativeToHexString(TEST_ERC20, CHAIN_ID_ETH) || "")
    )) || ""
  );
  const recipient = await Token.getAssociatedTokenAddress(
    ASSOCIATED_TOKEN_PROGRAM_ID,
    TOKEN_PROGRAM_ID,
    solanaMintKey,
    keypair.publicKey
  );
  // create the associated token account if it doesn't exist
  const associatedAddressInfo = await connection.getAccountInfo(recipient);
  if (!associatedAddressInfo) {
    const transaction = new Transaction().add(
      await Token.createAssociatedTokenAccountInstruction(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        solanaMintKey,
        recipient,
        keypair.publicKey, // owner
        keypair.publicKey // payer
      )
    );
    const { blockhash } = await connection.getRecentBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = keypair.publicKey;
    // sign, send, and confirm transaction
    transaction.partialSign(keypair);
    const txid = await connection.sendRawTransaction(transaction.serialize());
    await connection.confirmTransaction(txid);
  }
  // create a signer for Eth
  const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
  const amount = parseUnits("1", 18);
  // approve the bridge to spend tokens
  await approveEth(ETH_TOKEN_BRIDGE_ADDRESS, TEST_ERC20, signer, amount);
  // transfer tokens
  const receipt = await transferFromEth(
    ETH_TOKEN_BRIDGE_ADDRESS,
    signer,
    TEST_ERC20,
    amount,
    CHAIN_ID_SOLANA,
    hexToUint8Array(
      nativeToHexString(recipient.toString(), CHAIN_ID_SOLANA) || ""
    )
  );
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = await parseSequenceFromLogEth(
    receipt,
    ETH_CORE_BRIDGE_ADDRESS
  );
  provider.destroy();
  return sequence;
}
