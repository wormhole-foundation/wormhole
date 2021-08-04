import Wallet from "@project-serum/sol-wallet-adapter";
import { Token, TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  AccountMeta,
  Connection,
  PublicKey,
  SystemProgram,
  Transaction,
  TransactionInstruction,
} from "@solana/web3.js";
import { ethers } from "ethers";
import { arrayify, formatUnits, parseUnits, zeroPad } from "ethers/lib/utils";
import {
  Bridge__factory,
  Implementation__factory,
  TokenImplementation__factory,
} from "../ethers-contracts";
import { getSignedVAA } from "../sdk";
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

// TODO: this should probably be extended from the context somehow so that the signatures match
// TODO: allow for / handle cancellation?
// TODO: overall better input checking and error handling
export function transferFromEth(
  provider: ethers.providers.Web3Provider | undefined,
  tokenAddress: string,
  amount: string,
  recipientChain: ChainId,
  recipientAddress: Uint8Array | undefined
) {
  if (!provider || !recipientAddress) return;
  const signer = provider.getSigner();
  if (!signer) return;
  //TODO: check if token attestation exists on the target chain
  //TODO: don't hardcode, fetch decimals / share them with balance, how do we determine recipient chain?
  //TODO: more catches
  const amountParsed = parseUnits(amount, 18);
  (async () => {
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
    console.log("Amount:", formatUnits(amountParsed, 18));
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
      sequence
    );
    console.log("SIGNED VAA:", vaaBytes);
  })();
}

// TODO: should we share this with client? ooh, should client use the SDK ;)
// begin from clients\solana\main.ts
function ixFromRust(data: any): TransactionInstruction {
  let keys: Array<AccountMeta> = data.accounts.map(accountMetaFromRust);
  return new TransactionInstruction({
    programId: new PublicKey(data.program_id),
    data: Buffer.from(data.data),
    keys: keys,
  });
}

function accountMetaFromRust(meta: any): AccountMeta {
  return {
    pubkey: new PublicKey(meta.pubkey),
    isSigner: meta.is_signer,
    isWritable: meta.is_writable,
  };
}
// end from clients\solana\main.ts

// TODO: need to check transfer native vs transfer wrapped
// TODO: switch out targetProvider for generic address (this likely involves getting these in their respective contexts)
export function transferFromSolana(
  wallet: Wallet | undefined,
  payerAddress: string | undefined, //TODO: we may not need this since we have wallet
  fromAddress: string | undefined,
  mintAddress: string,
  amount: string,
  decimals: number,
  targetProvider: ethers.providers.Web3Provider | undefined,
  targetChain: ChainId
) {
  if (
    !wallet ||
    !wallet.publicKey ||
    !payerAddress ||
    !fromAddress ||
    !targetProvider
  )
    return;
  const targetSigner = targetProvider.getSigner();
  if (!targetSigner) return;
  (async () => {
    const targetAddressStr = await targetSigner.getAddress();
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
    const bridgeStatePK = new PublicKey(
      bridge.state_address(SOL_BRIDGE_ADDRESS)
    );
    const connection = new Connection(SOLANA_HOST, "confirmed");
    const bridgeStateAccountInfo = await connection.getAccountInfo(
      bridgeStatePK
    );
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
    const { transfer_native_ix, approval_authority_address } = await import(
      "token-bridge"
    );
    const approvalIx = Token.createApproveInstruction(
      TOKEN_PROGRAM_ID,
      new PublicKey(fromAddress),
      new PublicKey(approval_authority_address(SOL_TOKEN_BRIDGE_ADDRESS)),
      new PublicKey(payerAddress),
      [],
      Number(amountParsed)
    );
    const ix = ixFromRust(
      transfer_native_ix(
        SOL_TOKEN_BRIDGE_ADDRESS,
        SOL_BRIDGE_ADDRESS,
        payerAddress,
        fromAddress,
        mintAddress,
        nonce,
        amountParsed,
        fee,
        targetAddress,
        targetChain
      )
    );
    console.log(ix);
    const transaction = new Transaction().add(transferIx, approvalIx, ix);
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
  })();
}

const transferFrom = {
  [CHAIN_ID_ETH]: transferFromEth,
  [CHAIN_ID_SOLANA]: transferFromSolana,
};

export default transferFrom;
