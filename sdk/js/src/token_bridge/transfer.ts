import { Token, TOKEN_PROGRAM_ID } from "@solana/spl-token";
import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { ethers } from "ethers";
import {
  Bridge__factory,
  TokenImplementation__factory,
} from "../ethers-contracts";
import { getBridgeFeeIx, ixFromRust } from "../solana";
import { ChainId, CHAIN_ID_SOLANA, createNonce } from "../utils";

export async function getAllowanceEth(
  tokenBridgeAddress: string,
  tokenAddress: string,
  signer: ethers.Signer
) {
  const token = TokenImplementation__factory.connect(tokenAddress, signer);
  const signerAddress = await signer.getAddress();
  const allowance = await token.allowance(signerAddress, tokenBridgeAddress);

  return allowance;
}

export async function approveEth(
  tokenBridgeAddress: string,
  tokenAddress: string,
  signer: ethers.Signer,
  amount: ethers.BigNumberish
) {
  const token = TokenImplementation__factory.connect(tokenAddress, signer);
  return await (await token.approve(tokenBridgeAddress, amount)).wait();
}

export async function transferFromEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  tokenAddress: string,
  amount: ethers.BigNumberish,
  recipientChain: ChainId,
  recipientAddress: Uint8Array
) {
  //TODO: should we check if token attestation exists on the target chain
  const token = TokenImplementation__factory.connect(tokenAddress, signer);
  const fee = 0; // for now, this won't do anything, we may add later
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.transferTokens(
    tokenAddress,
    amount,
    recipientChain,
    recipientAddress,
    fee,
    createNonce()
  );
  const receipt = await v.wait();
  return receipt;
}

export async function transferFromTerra(
  walletAddress: string,
  tokenBridgeAddress: string,
  tokenAddress: string,
  amount: string,
  recipientChain: ChainId,
  recipientAddress: Uint8Array
) {
  const nonce = Math.round(Math.random() * 100000);
  return [
    new MsgExecuteContract(
      walletAddress,
      tokenAddress,
      {
        increase_allowance: {
          spender: tokenBridgeAddress,
          amount: amount,
          expires: {
            never: {},
          },
        },
      },
      { uluna: 10000 }
    ),
    new MsgExecuteContract(
      walletAddress,
      tokenBridgeAddress,
      {
        initiate_transfer: {
          asset: tokenAddress,
          amount: amount,
          recipient_chain: recipientChain,
          recipient: Buffer.from(recipientAddress).toString("base64"),
          fee: "0",
          nonce: nonce,
        },
      },
      { uluna: 10000 }
    ),
  ];
}

export async function transferFromSolana(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  fromAddress: string,
  mintAddress: string,
  amount: BigInt,
  targetAddress: Uint8Array,
  targetChain: ChainId,
  originAddress?: Uint8Array,
  originChain?: ChainId
) {
  const nonce = createNonce().readUInt32LE(0);
  const fee = BigInt(0); // for now, this won't do anything, we may add later
  const transferIx = await getBridgeFeeIx(
    connection,
    bridgeAddress,
    payerAddress
  );
  const {
    transfer_native_ix,
    transfer_wrapped_ix,
    approval_authority_address,
  } = await import("../solana/token/token_bridge");
  const approvalIx = Token.createApproveInstruction(
    TOKEN_PROGRAM_ID,
    new PublicKey(fromAddress),
    new PublicKey(approval_authority_address(tokenBridgeAddress)),
    new PublicKey(payerAddress),
    [],
    Number(amount)
  );
  let messageKey = Keypair.generate();
  const isSolanaNative =
    originChain === undefined || originChain === CHAIN_ID_SOLANA;
  if (!isSolanaNative && !originAddress) {
    throw new Error("originAddress is required when specifying originChain");
  }
  const ix = ixFromRust(
    isSolanaNative
      ? transfer_native_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          messageKey.publicKey.toString(),
          fromAddress,
          mintAddress,
          nonce,
          amount,
          fee,
          targetAddress,
          targetChain
        )
      : transfer_wrapped_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          messageKey.publicKey.toString(),
          fromAddress,
          payerAddress,
          originChain as number, // checked by isSolanaNative
          originAddress as Uint8Array, // checked by throw
          nonce,
          amount,
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
  return transaction;
}
