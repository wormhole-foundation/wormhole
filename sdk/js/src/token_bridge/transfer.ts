import { AccountLayout, Token, TOKEN_PROGRAM_ID, u64 } from "@solana/spl-token";
import {
  Connection,
  Keypair,
  PublicKey,
  SystemProgram,
  Transaction,
} from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { BigNumber, ethers } from "ethers";
import { isNativeDenom } from "..";
import {
  Bridge__factory,
  TokenImplementation__factory,
} from "../ethers-contracts";
import { getBridgeFeeIx, ixFromRust } from "../solana";
import { ChainId, CHAIN_ID_SOLANA, createNonce, WSOL_ADDRESS } from "../utils";

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

export async function transferFromEthNative(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  amount: ethers.BigNumberish,
  recipientChain: ChainId,
  recipientAddress: Uint8Array
) {
  const fee = 0; // for now, this won't do anything, we may add later
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.wrapAndTransferETH(
    recipientChain,
    recipientAddress,
    fee,
    createNonce(),
    {
      value: amount,
    }
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
  const isNativeAsset = isNativeDenom(tokenAddress);
  return isNativeAsset
    ? [
        new MsgExecuteContract(
          walletAddress,
          tokenBridgeAddress,
          {
            deposit_tokens: {},
          },
          { [tokenAddress]: amount }
        ),
        new MsgExecuteContract(
          walletAddress,
          tokenBridgeAddress,
          {
            initiate_transfer: {
              asset: {
                amount,
                info: {
                  native_token: {
                    denom: tokenAddress,
                  },
                },
              },
              recipient_chain: recipientChain,
              recipient: Buffer.from(recipientAddress).toString("base64"),
              fee: "0",
              nonce: nonce,
            },
          },
          {}
        ),
      ]
    : [
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
          {}
        ),
        new MsgExecuteContract(
          walletAddress,
          tokenBridgeAddress,
          {
            initiate_transfer: {
              asset: {
                amount: amount,
                info: {
                  token: {
                    contract_addr: tokenAddress,
                  },
                },
              },
              recipient_chain: recipientChain,
              recipient: Buffer.from(recipientAddress).toString("base64"),
              fee: "0",
              nonce: nonce,
            },
          },
          {}
        ),
      ];
}

export async function transferNativeSol(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  amount: BigInt,
  targetAddress: Uint8Array,
  targetChain: ChainId
) {
  //https://github.com/solana-labs/solana-program-library/blob/master/token/js/client/token.js
  const rentBalance = await Token.getMinBalanceRentForExemptAccount(connection);
  const mintPublicKey = new PublicKey(WSOL_ADDRESS);
  const payerPublicKey = new PublicKey(payerAddress);
  const ancillaryKeypair = Keypair.generate();

  //This will create a temporary account where the wSOL will be created.
  const createAncillaryAccountIx = SystemProgram.createAccount({
    fromPubkey: payerPublicKey,
    newAccountPubkey: ancillaryKeypair.publicKey,
    lamports: rentBalance, //spl token accounts need rent exemption
    space: AccountLayout.span,
    programId: TOKEN_PROGRAM_ID,
  });

  //Send in the amount of SOL which we want converted to wSOL
  const initialBalanceTransferIx = SystemProgram.transfer({
    fromPubkey: payerPublicKey,
    lamports: Number(amount),
    toPubkey: ancillaryKeypair.publicKey,
  });
  //Initialize the account as a WSOL account, with the original payerAddress as owner
  const initAccountIx = await Token.createInitAccountInstruction(
    TOKEN_PROGRAM_ID,
    mintPublicKey,
    ancillaryKeypair.publicKey,
    payerPublicKey
  );

  //Normal approve & transfer instructions, except that the wSOL is sent from the ancillary account.
  const { transfer_native_ix, approval_authority_address } = await import(
    "../solana/token/token_bridge"
  );
  const nonce = createNonce().readUInt32LE(0);
  const fee = BigInt(0); // for now, this won't do anything, we may add later
  const transferIx = await getBridgeFeeIx(
    connection,
    bridgeAddress,
    payerAddress
  );
  const approvalIx = Token.createApproveInstruction(
    TOKEN_PROGRAM_ID,
    ancillaryKeypair.publicKey,
    new PublicKey(approval_authority_address(tokenBridgeAddress)),
    payerPublicKey, //owner
    [],
    new u64(amount.toString(16), 16)
  );
  let messageKey = Keypair.generate();

  const ix = ixFromRust(
    transfer_native_ix(
      tokenBridgeAddress,
      bridgeAddress,
      payerAddress,
      messageKey.publicKey.toString(),
      ancillaryKeypair.publicKey.toString(),
      WSOL_ADDRESS,
      nonce,
      amount,
      fee,
      targetAddress,
      targetChain
    )
  );

  //Close the ancillary account for cleanup. Payer address receives any remaining funds
  const closeAccountIx = Token.createCloseAccountInstruction(
    TOKEN_PROGRAM_ID,
    ancillaryKeypair.publicKey, //account to close
    payerPublicKey, //Remaining funds destination
    payerPublicKey, //authority
    []
  );

  const { blockhash } = await connection.getRecentBlockhash();
  const transaction = new Transaction();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  transaction.add(createAncillaryAccountIx);
  transaction.add(initialBalanceTransferIx);
  transaction.add(initAccountIx);
  transaction.add(transferIx, approvalIx, ix);
  transaction.add(closeAccountIx);
  transaction.partialSign(messageKey);
  transaction.partialSign(ancillaryKeypair);
  return transaction;
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
  originChain?: ChainId,
  fromOwnerAddress?: string
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
    new PublicKey(fromOwnerAddress || payerAddress),
    [],
    new u64(amount.toString(16), 16)
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
          fromOwnerAddress || payerAddress,
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
