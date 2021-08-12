import Wallet from "@project-serum/sol-wallet-adapter";
import { Token, TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  Connection,
  Keypair,
  PublicKey,
  SystemProgram,
  Transaction,
} from "@solana/web3.js";
import { ethers } from "ethers";
import { arrayify, formatUnits, parseUnits, zeroPad } from "ethers/lib/utils";
import {
  Bridge__factory,
  Implementation__factory,
  TokenImplementation__factory,
} from "../ethers-contracts";
import { getSignedVAA, ixFromRust } from "../sdk";
import { hexToUint8Array } from "./array";
import {
  ChainId,
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
export async function transferFromEth(
  provider: ethers.providers.Web3Provider | undefined,
  signer: ethers.Signer | undefined,
  tokenAddress: string,
  decimals: number,
  amount: string,
  recipientChain: ChainId,
  recipientAddress: Uint8Array | undefined
) {
  if (!provider || !signer || !recipientAddress) return;
  //TODO: check if token attestation exists on the target chain
  //TODO: don't hardcode, fetch decimals / share them with balance, how do we determine recipient chain?
  //TODO: more catches
  const amountParsed = parseUnits(amount, decimals);
  const signerAddress = await signer.getAddress();
  console.log("Signer:", signerAddress);
  console.log("Token:", tokenAddress);
  const token = TokenImplementation__factory.connect(tokenAddress, signer);
  const allowance = await token.allowance(
    signerAddress,
    ETH_TOKEN_BRIDGE_ADDRESS
  );
  console.log("Allowance", allowance.toString()); //TODO: should we check that this is zero and warn if it isn't?
  const transaction = await token.approve(
    ETH_TOKEN_BRIDGE_ADDRESS,
    amountParsed
  );
  console.log(transaction);
  const fee = 0; // for now, this won't do anything, we may add later
  const nonceConst = Math.random() * 100000;
  const nonceBuffer = Buffer.alloc(4);
  nonceBuffer.writeUInt32LE(nonceConst, 0);
  console.log("Initiating transfer");
  console.log("Amount:", formatUnits(amountParsed, decimals));
  console.log("To chain:", recipientChain);
  console.log("To address:", recipientAddress);
  console.log("Fees:", fee);
  console.log("Nonce:", nonceBuffer);
  const bridge = Bridge__factory.connect(ETH_TOKEN_BRIDGE_ADDRESS, signer);
  const v = await bridge.transferTokens(
    tokenAddress,
    amountParsed,
    recipientChain,
    recipientAddress,
    fee,
    nonceBuffer
  );
  const receipt = await v.wait();
  // TODO: log parsing should be part of a utility
  // TODO: dangerous!(?)
  const bridgeLog = receipt.logs.filter((l) => {
    console.log(l.address, ETH_BRIDGE_ADDRESS);
    return l.address === ETH_BRIDGE_ADDRESS;
  })[0];
  const {
    args: { sender, sequence },
  } = Implementation__factory.createInterface().parseLog(bridgeLog);
  console.log(sender, sequence);
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
export async function transferFromSolana(
  wallet: Wallet | undefined,
  payerAddress: string | undefined, //TODO: we may not need this since we have wallet
  fromAddress: string | undefined,
  mintAddress: string,
  amount: string,
  decimals: number,
  targetAddressStr: string | undefined,
  targetChain: ChainId,
  originAddress?: string,
  originChain?: ChainId
) {
  if (
    !wallet ||
    !wallet.publicKey ||
    !payerAddress ||
    !fromAddress ||
    !targetAddressStr ||
    (originChain && !originAddress)
  )
    return;
  const targetAddress = zeroPad(arrayify(targetAddressStr), 32);
  const nonceConst = Math.random() * 100000;
  const nonceBuffer = Buffer.alloc(4);
  nonceBuffer.writeUInt32LE(nonceConst, 0);
  const nonce = nonceBuffer.readUInt32LE(0);
  const amountParsed = parseUnits(amount, decimals).toBigInt();
  const fee = BigInt(0); // for now, this won't do anything, we may add later
  console.log("program:", SOL_TOKEN_BRIDGE_ADDRESS);
  console.log("bridge:", SOL_BRIDGE_ADDRESS);
  console.log("payer:", payerAddress);
  console.log("from:", fromAddress);
  console.log("token:", mintAddress);
  console.log("nonce:", nonce);
  console.log("amount:", amountParsed);
  console.log("fee:", fee);
  console.log("target:", targetAddressStr, targetAddress);
  console.log("chain:", targetChain);
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
  const {
    transfer_native_ix,
    transfer_wrapped_ix,
    approval_authority_address,
    emitter_address,
  } = await import("token-bridge");
  const approvalIx = Token.createApproveInstruction(
    TOKEN_PROGRAM_ID,
    new PublicKey(fromAddress),
    new PublicKey(approval_authority_address(SOL_TOKEN_BRIDGE_ADDRESS)),
    new PublicKey(payerAddress),
    [],
    Number(amountParsed)
  );

  let messageKey = Keypair.generate();
  const isSolanaNative =
    originChain === undefined || originChain === CHAIN_ID_SOLANA;
  console.log(isSolanaNative ? "SENDING NATIVE" : "SENDING WRAPPED");
  const ix = ixFromRust(
    isSolanaNative
      ? transfer_native_ix(
          SOL_TOKEN_BRIDGE_ADDRESS,
          SOL_BRIDGE_ADDRESS,
          payerAddress,
          messageKey.publicKey.toString(),
          fromAddress,
          mintAddress,
          nonce,
          amountParsed,
          fee,
          targetAddress,
          targetChain
        )
      : transfer_wrapped_ix(
          SOL_TOKEN_BRIDGE_ADDRESS,
          SOL_BRIDGE_ADDRESS,
          payerAddress,
          messageKey.publicKey.toString(),
          fromAddress,
          payerAddress,
          originChain as number, // checked by isSolanaNative
          zeroPad(hexToUint8Array(originAddress as string), 32), // checked by initial check
          nonce,
          amountParsed,
          fee,
          targetAddress,
          targetChain
        )
  );
  const transaction = new Transaction().add(transferIx, approvalIx, ix);
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
    sequence
  );
  console.log("SIGNED VAA:", vaaBytes);
  return vaaBytes;
}

const transferFrom = {
  [CHAIN_ID_ETH]: transferFromEth,
  [CHAIN_ID_SOLANA]: transferFromSolana,
};

export default transferFrom;
