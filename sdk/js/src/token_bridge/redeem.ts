import { AccountLayout, Token, TOKEN_PROGRAM_ID, u64 } from "@solana/spl-token";
import {
  Connection,
  Keypair,
  PublicKey,
  SystemProgram,
  Transaction,
} from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { Algodv2 } from "algosdk";
import { ethers, Overrides } from "ethers";
import { fromUint8Array } from "js-base64";
import { TransactionSignerPair, _submitVAAAlgorand } from "../algorand";
import { Bridge__factory } from "../ethers-contracts";
import { ixFromRust } from "../solana";
import { importCoreWasm, importTokenWasm } from "../solana/wasm";
import {
  CHAIN_ID_NEAR,
  CHAIN_ID_SOLANA,
  ChainId,
  MAX_VAA_DECIMALS,
  WSOL_ADDRESS,
  WSOL_DECIMALS,
  uint8ArrayToHex,
  callFunctionNear,
  hashLookup
} from "../utils";

import { getForeignAssetNear } from ".";

import { _parseVAAAlgorand } from "../algorand";

import { hexToNativeString } from "../utils/array";
import { parseTransferPayload } from "../utils/parseVaa";
import BN from "bn.js";
import { MsgExecuteContract as MsgExecuteContractInjective } from "@injectivelabs/sdk-ts";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { Provider } from "near-api-js/lib/providers";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";
import { AptosClient, Types } from "aptos";
import { completeTransfer as completeTransferAptos } from "../aptos";

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
    msg: {
      data: fromUint8Array(signedVAA),
    },
    action: "submit_vaa",
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
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  signedVAA: Uint8Array
) {
  const { parse_vaa } = await importCoreWasm();
  const { complete_transfer_native_ix } = await importTokenWasm();
  const parsedVAA = parse_vaa(signedVAA);
  const parsedPayload = parseTransferPayload(
    Buffer.from(new Uint8Array(parsedVAA.payload))
  );
  const targetAddress = hexToNativeString(
    parsedPayload.targetAddress,
    CHAIN_ID_SOLANA
  );
  if (!targetAddress) {
    throw new Error("Failed to read the target address.");
  }
  const targetPublicKey = new PublicKey(targetAddress);
  const targetAmount =
    parsedPayload.amount *
    BigInt(WSOL_DECIMALS - MAX_VAA_DECIMALS) *
    BigInt(10);
  const rentBalance = await Token.getMinBalanceRentForExemptAccount(connection);
  const mintPublicKey = new PublicKey(WSOL_ADDRESS);
  const payerPublicKey = new PublicKey(payerAddress);
  const ancillaryKeypair = Keypair.generate();

  const completeTransferIx = ixFromRust(
    complete_transfer_native_ix(
      tokenBridgeAddress,
      bridgeAddress,
      payerAddress,
      signedVAA
    )
  );

  //This will create a temporary account where the wSOL will be moved
  const createAncillaryAccountIx = SystemProgram.createAccount({
    fromPubkey: payerPublicKey,
    newAccountPubkey: ancillaryKeypair.publicKey,
    lamports: rentBalance, //spl token accounts need rent exemption
    space: AccountLayout.span,
    programId: TOKEN_PROGRAM_ID,
  });

  //Initialize the account as a WSOL account, with the original payerAddress as owner
  const initAccountIx = await Token.createInitAccountInstruction(
    TOKEN_PROGRAM_ID,
    mintPublicKey,
    ancillaryKeypair.publicKey,
    payerPublicKey
  );

  //Send in the amount of wSOL which we want converted to SOL
  const balanceTransferIx = Token.createTransferInstruction(
    TOKEN_PROGRAM_ID,
    targetPublicKey,
    ancillaryKeypair.publicKey,
    payerPublicKey,
    [],
    new u64(targetAmount.toString(16), 16)
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
  transaction.add(completeTransferIx);
  transaction.add(createAncillaryAccountIx);
  transaction.add(initAccountIx);
  transaction.add(balanceTransferIx);
  transaction.add(closeAccountIx);
  transaction.partialSign(ancillaryKeypair);
  return transaction;
}

export async function redeemOnSolana(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  signedVAA: Uint8Array,
  feeRecipientAddress?: string
) {
  const { parse_vaa } = await importCoreWasm();
  const parsedVAA = parse_vaa(signedVAA);
  const isSolanaNative =
    Buffer.from(new Uint8Array(parsedVAA.payload)).readUInt16BE(65) ===
    CHAIN_ID_SOLANA;
  const { complete_transfer_wrapped_ix, complete_transfer_native_ix } =
    await importTokenWasm();
  const ixs = [];
  if (isSolanaNative) {
    ixs.push(
      ixFromRust(
        complete_transfer_native_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          signedVAA,
          feeRecipientAddress
        )
      )
    );
  } else {
    ixs.push(
      ixFromRust(
        complete_transfer_wrapped_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          signedVAA,
          feeRecipientAddress
        )
      )
    );
  }
  const transaction = new Transaction().add(...ixs);
  const { blockhash } = await connection.getRecentBlockhash();
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

export function redeemFromAptos(
  client: AptosClient,
  tokenBridgeAddress: string,
  transferVAA: Uint8Array,
  feeRecipientAddress: string
): Promise<Types.EntryFunctionPayload> {
  // TODO: show we use completeTransferAndRegister here
  return completeTransferAptos(
    client,
    tokenBridgeAddress,
    transferVAA,
    feeRecipientAddress
  );
}
