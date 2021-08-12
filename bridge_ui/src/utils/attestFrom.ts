import Wallet from "@project-serum/sol-wallet-adapter";
import {
  Connection,
  Keypair,
  PublicKey,
  SystemProgram,
  Transaction,
} from "@solana/web3.js";
import { ethers } from "ethers";
import { arrayify, zeroPad } from "ethers/lib/utils";
import { Bridge__factory, Implementation__factory } from "../ethers-contracts";
import { getSignedVAA, ixFromRust } from "../sdk";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  ETH_BRIDGE_ADDRESS,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "./consts";

// TODO: allow for / handle cancellation?
// TODO: overall better input checking and error handling
export async function attestFromEth(
  provider: ethers.providers.Web3Provider | undefined,
  signer: ethers.Signer | undefined,
  tokenAddress: string
) {
  if (!provider || !signer) return;
  //TODO: more catches
  const signerAddress = await signer.getAddress();
  console.log("Signer:", signerAddress);
  console.log("Token:", tokenAddress);
  const nonceConst = Math.random() * 100000;
  const nonceBuffer = Buffer.alloc(4);
  nonceBuffer.writeUInt32LE(nonceConst, 0);
  console.log("Initiating attestation");
  console.log("Nonce:", nonceBuffer);
  const bridge = Bridge__factory.connect(ETH_TOKEN_BRIDGE_ADDRESS, signer);
  const v = await bridge.attestToken(tokenAddress, nonceBuffer);
  const receipt = await v.wait();
  // TODO: log parsing should be part of a utility
  // TODO: dangerous!(?)
  const bridgeLog = receipt.logs.filter((l) => {
    console.log(l.address, ETH_BRIDGE_ADDRESS);
    return l.address === ETH_BRIDGE_ADDRESS;
  })[0];
  const {
    args: { sequence },
  } = Implementation__factory.createInterface().parseLog(bridgeLog);
  console.log("SEQ:", sequence);
  const emitterAddress = Buffer.from(
    zeroPad(arrayify(ETH_TOKEN_BRIDGE_ADDRESS), 32)
  ).toString("hex");
  const { vaaBytes } = await getSignedVAA(
    CHAIN_ID_ETH,
    emitterAddress,
    sequence.toString()
  );
  console.log("SIGNED VAA:", vaaBytes);
  return vaaBytes;
}

// TODO: need to check transfer native vs transfer wrapped
// TODO: switch out targetProvider for generic address (this likely involves getting these in their respective contexts)
export async function attestFromSolana(
  wallet: Wallet | undefined,
  payerAddress: string | undefined, //TODO: we may not need this since we have wallet
  mintAddress: string
) {
  if (!wallet || !wallet.publicKey || !payerAddress) return;
  const nonceConst = Math.random() * 100000;
  const nonceBuffer = Buffer.alloc(4);
  nonceBuffer.writeUInt32LE(nonceConst, 0);
  const nonce = nonceBuffer.readUInt32LE(0);
  console.log("program:", SOL_TOKEN_BRIDGE_ADDRESS);
  console.log("bridge:", SOL_BRIDGE_ADDRESS);
  console.log("payer:", payerAddress);
  console.log("token:", mintAddress);
  console.log("nonce:", nonce);
  const bridge = await import("bridge");
  const feeAccount = await bridge.fee_collector_address(SOL_BRIDGE_ADDRESS);
  const bridgeStatePK = new PublicKey(bridge.state_address(SOL_BRIDGE_ADDRESS));
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const bridgeStateAccountInfo = await connection.getAccountInfo(bridgeStatePK);
  if (bridgeStateAccountInfo?.data === undefined) {
    throw new Error("bridge state not found");
  }
  const bridgeState = bridge.parse_state(
    new Uint8Array(bridgeStateAccountInfo?.data)
  );
  const transferIx = SystemProgram.transfer({
    fromPubkey: new PublicKey(payerAddress),
    toPubkey: new PublicKey(feeAccount),
    lamports: bridgeState.config.fee,
  });
  // TODO: pass in connection
  // Add transfer instruction to transaction
  const { attest_ix, emitter_address } = await import("token-bridge");
  const messageKey = Keypair.generate();
  const ix = ixFromRust(
    attest_ix(
      SOL_TOKEN_BRIDGE_ADDRESS,
      SOL_BRIDGE_ADDRESS,
      payerAddress,
      messageKey.publicKey.toString(),
      mintAddress,
      nonce
    )
  );
  const transaction = new Transaction().add(transferIx, ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  transaction.partialSign(messageKey);
  // Sign transaction, broadcast, and confirm
  const signed = await wallet.signTransaction(transaction);
  console.log("SIGNED", signed);
  const txid = await connection.sendRawTransaction(signed.serialize());
  console.log("SENT", txid);
  const conf = await connection.confirmTransaction(txid);
  console.log("CONFIRMED", conf);
  const info = await connection.getTransaction(txid);
  console.log("INFO", info);
  // TODO: log parsing should be part of a utility
  // TODO: better parsing, safer
  const SEQ_LOG = "Program log: Sequence: ";
  const sequence = info?.meta?.logMessages
    ?.filter((msg) => msg.startsWith(SEQ_LOG))[0]
    .replace(SEQ_LOG, "");
  if (!sequence) {
    throw new Error("sequence not found");
  }
  console.log("SEQ", sequence);
  const emitterAddress = Buffer.from(
    zeroPad(
      new PublicKey(emitter_address(SOL_TOKEN_BRIDGE_ADDRESS)).toBytes(),
      32
    )
  ).toString("hex");
  const { vaaBytes } = await getSignedVAA(
    CHAIN_ID_SOLANA,
    emitterAddress,
    sequence.toString()
  );
  console.log("SIGNED VAA:", vaaBytes);
  return vaaBytes;
}

const attestFrom = {
  [CHAIN_ID_ETH]: attestFromEth,
  [CHAIN_ID_SOLANA]: attestFromSolana,
};

export default attestFrom;
