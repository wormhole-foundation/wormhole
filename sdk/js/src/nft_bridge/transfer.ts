import { Token, TOKEN_PROGRAM_ID } from "@solana/spl-token";
import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { ethers } from "ethers";
import {
  NFTBridge__factory,
  NFTImplementation__factory,
} from "../ethers-contracts";
import { getBridgeFeeIx, ixFromRust } from "../solana";
import { importNftWasm } from "../solana/wasm";
import { ChainId, CHAIN_ID_SOLANA, createNonce } from "../utils";

export async function transferFromEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  tokenAddress: string,
  tokenID: ethers.BigNumberish,
  recipientChain: ChainId,
  recipientAddress: Uint8Array
): Promise<ethers.ContractReceipt> {
  //TODO: should we check if token attestation exists on the target chain
  const token = NFTImplementation__factory.connect(tokenAddress, signer);
  await (await token.approve(tokenBridgeAddress, tokenID)).wait();
  const bridge = NFTBridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.transferNFT(
    tokenAddress,
    tokenID,
    recipientChain,
    recipientAddress,
    createNonce()
  );
  const receipt = await v.wait();
  return receipt;
}

export async function transferFromSolana(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  fromAddress: string,
  mintAddress: string,
  targetAddress: Uint8Array,
  targetChain: ChainId,
  originAddress?: Uint8Array,
  originChain?: ChainId,
  originTokenId?: Uint8Array
): Promise<Transaction> {
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await getBridgeFeeIx(
    connection,
    bridgeAddress,
    payerAddress
  );
  const {
    transfer_native_ix,
    transfer_wrapped_ix,
    approval_authority_address,
  } = await importNftWasm();
  const approvalIx = Token.createApproveInstruction(
    TOKEN_PROGRAM_ID,
    new PublicKey(fromAddress),
    new PublicKey(approval_authority_address(tokenBridgeAddress)),
    new PublicKey(payerAddress),
    [],
    Number(1)
  );
  let messageKey = Keypair.generate();
  const isSolanaNative =
    originChain === undefined || originChain === CHAIN_ID_SOLANA;
  if (!isSolanaNative && !originAddress && !originTokenId) {
    throw new Error(
      "originAddress and originTokenId are required when specifying originChain"
    );
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
          originTokenId as Uint8Array, // checked by throw
          nonce,
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

export async function transferFromTerra(
  walletAddress: string,
  tokenBridgeAddress: string,
  tokenAddress: string,
  tokenID: string,
  recipientChain: ChainId,
  recipientAddress: Uint8Array
): Promise<MsgExecuteContract[]> {
  const nonce = Math.round(Math.random() * 100000);
  return [
    new MsgExecuteContract(
      walletAddress,
      tokenAddress,
      {
        approve: {
          spender: tokenBridgeAddress,
          token_id: tokenID,
        },
      },
      {}
    ),
    new MsgExecuteContract(
      walletAddress,
      tokenBridgeAddress,
      {
        initiate_transfer: {
          contract_addr: tokenAddress,
          token_id: tokenID,
          recipient_chain: recipientChain,
          recipient: Buffer.from(recipientAddress).toString("base64"),
          nonce: nonce,
        },
      },
      {}
    ),
  ];
}
