import {
  ACCOUNT_SIZE,
  createCloseAccountInstruction,
  createInitializeAccountInstruction,
  createTransferInstruction,
  getMinimumBalanceForRentExemptAccount,
  getMint,
  NATIVE_MINT,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import {
  Commitment,
  Connection,
  Keypair,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  Transaction,
} from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { ethers, Overrides } from "ethers";
import { fromUint8Array } from "js-base64";
import BN from "bn.js";
import {
  TransactionSignerPair,
  _parseVAAAlgorand,
  _submitVAAAlgorand,
} from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import {
  CHAIN_ID_NEAR,
  CHAIN_ID_SOLANA,
  ChainId,
  MAX_VAA_DECIMALS,
  uint8ArrayToHex,
  callFunctionNear,
  hashLookup,
} from "../utils";
import { MsgExecuteContractCompat as MsgExecuteContractInjective } from "@injectivelabs/sdk-ts";
import {
  createCompleteTransferNativeInstruction,
  createCompleteTransferWrappedInstruction,
} from "../solana/tokenBridge";
import { SignedVaa, parseTokenTransferVaa } from "../vaa";
import { getForeignAssetNear } from "./getForeignAsset";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { Provider } from "near-api-js/lib/providers";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";
import { AptosClient, Types } from "aptos";
import { completeTransferAndRegister } from "../aptos";
import {
  JsonRpcProvider,
  SUI_CLOCK_OBJECT_ID,
  TransactionBlock,
} from "@mysten/sui.js";
import { getTokenCoinType, getObjectFields } from "../sui";

export async function redeemOnEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array,
  overrides: Overrides & { from?: string | Promise<string> } = {}
) {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.completeTransfer(signedVAA, overrides);
  const receipt = await v.wait();
  return receipt;
}

export async function redeemOnEthNative(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  signedVAA: Uint8Array,
  overrides: Overrides & { from?: string | Promise<string> } = {}
) {
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v = await bridge.completeTransferAndUnwrapETH(signedVAA, overrides);
  const receipt = await v.wait();
  return receipt;
}

export async function redeemOnTerra(
  tokenBridgeAddress: string,
  walletAddress: string,
  signedVAA: Uint8Array
) {
  return new MsgExecuteContract(walletAddress, tokenBridgeAddress, {
    submit_vaa: {
      data: fromUint8Array(signedVAA),
    },
  });
}

/**
 * Submits the supplied VAA to Injective
 * @param tokenBridgeAddress Address of Inj token bridge contract
 * @param walletAddress Address of wallet in inj format
 * @param signedVAA VAA with the attestation message
 * @returns Message to be broadcast
 */
export async function submitVAAOnInjective(
  tokenBridgeAddress: string,
  walletAddress: string,
  signedVAA: Uint8Array
): Promise<MsgExecuteContractInjective> {
  return MsgExecuteContractInjective.fromJSON({
    contractAddress: tokenBridgeAddress,
    sender: walletAddress,
    exec: {
      msg: {
        data: fromUint8Array(signedVAA),
      },
      action: "submit_vaa",
    },
  });
}
export const redeemOnInjective = submitVAAOnInjective;

export function redeemOnXpla(
  tokenBridgeAddress: string,
  walletAddress: string,
  signedVAA: Uint8Array
): XplaMsgExecuteContract {
  return new XplaMsgExecuteContract(walletAddress, tokenBridgeAddress, {
    submit_vaa: {
      data: fromUint8Array(signedVAA),
    },
  });
}

export async function redeemAndUnwrapOnSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  tokenBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  signedVaa: SignedVaa,
  commitment?: Commitment
) {
  const parsed = parseTokenTransferVaa(signedVaa);
  const targetPublicKey = new PublicKey(parsed.to);
  const targetAmount = await getMint(connection, NATIVE_MINT, commitment).then(
    (info) =>
      parsed.amount * BigInt(Math.pow(10, info.decimals - MAX_VAA_DECIMALS))
  );
  const rentBalance = await getMinimumBalanceForRentExemptAccount(
    connection,
    commitment
  );
  if (Buffer.compare(parsed.tokenAddress, NATIVE_MINT.toBuffer()) != 0) {
    return Promise.reject("tokenAddress != NATIVE_MINT");
  }
  const payerPublicKey = new PublicKey(payerAddress);
  const ancillaryKeypair = Keypair.generate();

  const completeTransferIx = createCompleteTransferNativeInstruction(
    tokenBridgeAddress,
    bridgeAddress,
    payerPublicKey,
    signedVaa
  );

  //This will create a temporary account where the wSOL will be moved
  const createAncillaryAccountIx = SystemProgram.createAccount({
    fromPubkey: payerPublicKey,
    newAccountPubkey: ancillaryKeypair.publicKey,
    lamports: rentBalance, //spl token accounts need rent exemption
    space: ACCOUNT_SIZE,
    programId: TOKEN_PROGRAM_ID,
  });

  //Initialize the account as a WSOL account, with the original payerAddress as owner
  const initAccountIx = createInitializeAccountInstruction(
    ancillaryKeypair.publicKey,
    NATIVE_MINT,
    payerPublicKey
  );

  //Send in the amount of wSOL which we want converted to SOL
  const balanceTransferIx = createTransferInstruction(
    targetPublicKey,
    ancillaryKeypair.publicKey,
    payerPublicKey,
    targetAmount.valueOf()
  );

  //Close the ancillary account for cleanup. Payer address receives any remaining funds
  const closeAccountIx = createCloseAccountInstruction(
    ancillaryKeypair.publicKey, //account to close
    payerPublicKey, //Remaining funds destination
    payerPublicKey //authority
  );

  const { blockhash } = await connection.getLatestBlockhash(commitment);
  const transaction = new Transaction();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = payerPublicKey;
  transaction.add(
    completeTransferIx,
    createAncillaryAccountIx,
    initAccountIx,
    balanceTransferIx,
    closeAccountIx
  );
  transaction.partialSign(ancillaryKeypair);
  return transaction;
}

export async function redeemOnSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  tokenBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  signedVaa: SignedVaa,
  feeRecipientAddress?: PublicKeyInitData,
  commitment?: Commitment
) {
  const parsed = parseTokenTransferVaa(signedVaa);
  const createCompleteTransferInstruction =
    parsed.tokenChain == CHAIN_ID_SOLANA
      ? createCompleteTransferNativeInstruction
      : createCompleteTransferWrappedInstruction;
  const transaction = new Transaction().add(
    createCompleteTransferInstruction(
      tokenBridgeAddress,
      bridgeAddress,
      payerAddress,
      parsed,
      feeRecipientAddress
    )
  );
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  return transaction;
}

/**
 * This basically just submits the VAA to Algorand
 * @param client AlgodV2 client
 * @param tokenBridgeId Token bridge ID
 * @param bridgeId Core bridge ID
 * @param vaa The VAA to be redeemed
 * @param acct Sending account
 * @returns Transaction ID(s)
 */
export async function redeemOnAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  bridgeId: bigint,
  vaa: Uint8Array,
  senderAddr: string
): Promise<TransactionSignerPair[]> {
  return await _submitVAAAlgorand(
    client,
    tokenBridgeId,
    bridgeId,
    vaa,
    senderAddr
  );
}

export async function redeemOnNear(
  provider: Provider,
  account: string,
  tokenBridge: string,
  vaa: Uint8Array
): Promise<FunctionCallOptions[]> {
  const options: FunctionCallOptions[] = [];
  const p = _parseVAAAlgorand(vaa);

  if (p.ToChain !== CHAIN_ID_NEAR) {
    throw new Error("Not destined for NEAR");
  }

  const { found, value: receiver } = await hashLookup(
    provider,
    tokenBridge,
    uint8ArrayToHex(p.ToAddress as Uint8Array)
  );

  if (!found) {
    throw new Error(
      "Unregistered receiver (receiving account is not registered)"
    );
  }

  const token = await getForeignAssetNear(
    provider,
    tokenBridge,
    p.FromChain as ChainId,
    p.Contract as string
  );

  if (
    (p.Contract as string) !==
    "0000000000000000000000000000000000000000000000000000000000000000"
  ) {
    if (token === "" || token === null) {
      throw new Error("Unregistered token (has it been attested?)");
    }

    const bal = await callFunctionNear(
      provider,
      token as string,
      "storage_balance_of",
      {
        account_id: receiver,
      }
    );

    if (bal === null) {
      options.push({
        contractId: token as string,
        methodName: "storage_deposit",
        args: { account_id: receiver, registration_only: true },
        gas: new BN("100000000000000"),
        attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
      });
    }

    if (
      p.Fee !== undefined &&
      Buffer.compare(
        p.Fee,
        Buffer.from(
          "0000000000000000000000000000000000000000000000000000000000000000",
          "hex"
        )
      ) !== 0
    ) {
      const bal = await callFunctionNear(
        provider,
        token as string,
        "storage_balance_of",
        {
          account_id: account,
        }
      );

      if (bal === null) {
        options.push({
          contractId: token as string,
          methodName: "storage_deposit",
          args: { account_id: account, registration_only: true },
          gas: new BN("100000000000000"),
          attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
        });
      }
    }
  }

  options.push({
    contractId: tokenBridge,
    methodName: "submit_vaa",
    args: {
      vaa: uint8ArrayToHex(vaa),
    },
    attachedDeposit: new BN("100000000000000000000000"),
    gas: new BN("150000000000000"),
  });

  options.push({
    contractId: tokenBridge,
    methodName: "submit_vaa",
    args: {
      vaa: uint8ArrayToHex(vaa),
    },
    attachedDeposit: new BN("100000000000000000000000"),
    gas: new BN("150000000000000"),
  });

  return options;
}

/**
 * Register the token specified in the given VAA in the transfer recipient's account if necessary
 * and complete the transfer.
 * @param client Client used to transfer data to/from Aptos node
 * @param tokenBridgeAddress Address of token bridge
 * @param transferVAA Bytes of transfer VAA
 * @returns Transaction payload
 */
export function redeemOnAptos(
  client: AptosClient,
  tokenBridgeAddress: string,
  transferVAA: Uint8Array
): Promise<Types.EntryFunctionPayload> {
  return completeTransferAndRegister(client, tokenBridgeAddress, transferVAA);
}

export async function redeemOnSui(
  provider: JsonRpcProvider,
  tokenBridgeAddress: string,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  transferVAA: Uint8Array
): Promise<TransactionBlock> {
  const parsed = parseTokenTransferVaa(transferVAA);
  const coinType = await getTokenCoinType(
    provider,
    tokenBridgeAddress,
    tokenBridgeStateObjectId,
    parsed.tokenAddress,
    parsed.tokenChain
  );
  if (coinType === null) {
    throw new Error("Unable to fetch coin type for token");
  }
  const tx = new TransactionBlock();
  tx.moveCall({
    target: `${tokenBridgeAddress}::token_bridge::complete_transfer`,
    arguments: [
      tx.object(tokenBridgeStateObjectId),
      tx.object(coreBridgeStateObjectId),
      tx.pure([...transferVAA]),
      tx.object(SUI_CLOCK_OBJECT_ID),
    ],
    typeArguments: [coinType],
  });
  return tx;
}
