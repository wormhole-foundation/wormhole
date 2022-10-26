import {
  AccountLayout,
  createCloseAccountInstruction,
  createInitializeAccountInstruction,
  getMinimumBalanceForRentExemptMint,
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
  Transaction as SolanaTransaction,
} from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { MsgExecuteContract as MsgExecuteContractInjective } from "@injectivelabs/sdk-ts";
import {
  Algodv2,
  bigIntToBytes,
  getApplicationAddress,
  makeApplicationCallTxnFromObject,
  makeAssetTransferTxnWithSuggestedParamsFromObject,
  makePaymentTxnWithSuggestedParamsFromObject,
  OnApplicationComplete,
  SuggestedParams,
  Transaction as AlgorandTransaction,
} from "algosdk";
import { ethers, Overrides, PayableOverrides } from "ethers";
import BN from "bn.js";
import { isNativeDenom } from "../terra";
import { getIsWrappedAssetNear } from "..";
import {
  assetOptinCheck,
  getMessageFee,
  optin,
  TransactionSignerPair,
} from "../algorand";
import { getEmitterAddressAlgorand } from "../bridge";
import {
  Bridge__factory,
  TokenImplementation__factory,
} from "../ethers-contracts";
import { createBridgeFeeTransferInstruction } from "../solana";
import {
  createApproveAuthoritySignerInstruction,
  createTransferNativeInstruction,
  createTransferNativeWithPayloadInstruction,
  createTransferWrappedInstruction,
  createTransferWrappedWithPayloadInstruction,
} from "../solana/tokenBridge";
import {
  ChainId,
  ChainName,
  coalesceChainId,
  createNonce,
  hexToUint8Array,
  safeBigIntToNumber,
  textToUint8Array,
  uint8ArrayToHex,
  CHAIN_ID_SOLANA,
  callFunctionNear,
} from "../utils";
import { isNativeDenomInjective, isNativeDenomXpla } from "../cosmwasm";
import { Types } from "aptos";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { Provider } from "near-api-js/lib/providers";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";
import {
  transferTokens as transferTokensAptos,
  transferTokensWithPayload,
} from "../aptos";

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
  amount: ethers.BigNumberish,
  overrides: Overrides & { from?: string | Promise<string> } = {}
) {
  const token = TokenImplementation__factory.connect(tokenAddress, signer);
  return await (
    await token.approve(tokenBridgeAddress, amount, overrides)
  ).wait();
}

export async function transferFromEth(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  tokenAddress: string,
  amount: ethers.BigNumberish,
  recipientChain: ChainId | ChainName,
  recipientAddress: Uint8Array,
  relayerFee: ethers.BigNumberish = 0,
  overrides: PayableOverrides & { from?: string | Promise<string> } = {},
  payload: Uint8Array | null = null
) {
  const recipientChainId = coalesceChainId(recipientChain);
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v =
    payload === null
      ? await bridge.transferTokens(
          tokenAddress,
          amount,
          recipientChainId,
          recipientAddress,
          relayerFee,
          createNonce(),
          overrides
        )
      : await bridge.transferTokensWithPayload(
          tokenAddress,
          amount,
          recipientChainId,
          recipientAddress,
          createNonce(),
          payload,
          overrides
        );
  const receipt = await v.wait();
  return receipt;
}

export async function transferFromEthNative(
  tokenBridgeAddress: string,
  signer: ethers.Signer,
  amount: ethers.BigNumberish,
  recipientChain: ChainId | ChainId,
  recipientAddress: Uint8Array,
  relayerFee: ethers.BigNumberish = 0,
  overrides: PayableOverrides & { from?: string | Promise<string> } = {},
  payload: Uint8Array | null = null
) {
  const recipientChainId = coalesceChainId(recipientChain);
  const bridge = Bridge__factory.connect(tokenBridgeAddress, signer);
  const v =
    payload === null
      ? await bridge.wrapAndTransferETH(
          recipientChainId,
          recipientAddress,
          relayerFee,
          createNonce(),
          {
            ...overrides,
            value: amount,
          }
        )
      : await bridge.wrapAndTransferETHWithPayload(
          recipientChainId,
          recipientAddress,
          createNonce(),
          payload,
          {
            ...overrides,
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
  recipientChain: ChainId | ChainName,
  recipientAddress: Uint8Array,
  relayerFee: string = "0",
  payload: Uint8Array | null = null
) {
  const recipientChainId = coalesceChainId(recipientChain);
  const nonce = Math.round(Math.random() * 100000);
  const isNativeAsset = isNativeDenom(tokenAddress);
  const mk_initiate_transfer = (info: object) =>
    payload
      ? {
          initiate_transfer_with_payload: {
            asset: {
              amount,
              info,
            },
            recipient_chain: recipientChainId,
            recipient: Buffer.from(recipientAddress).toString("base64"),
            fee: relayerFee,
            nonce: nonce,
            payload: payload,
          },
        }
      : {
          initiate_transfer: {
            asset: {
              amount,
              info,
            },
            recipient_chain: recipientChainId,
            recipient: Buffer.from(recipientAddress).toString("base64"),
            fee: relayerFee,
            nonce: nonce,
          },
        };
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
          mk_initiate_transfer({
            native_token: {
              denom: tokenAddress,
            },
          }),
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
          mk_initiate_transfer({
            token: {
              contract_addr: tokenAddress,
            },
          }),
          {}
        ),
      ];
}

/**
 * Creates the necessary messages to transfer an asset
 * @param walletAddress Address of the Inj wallet
 * @param tokenBridgeAddress Address of the token bridge contract
 * @param tokenAddress Address of the token being transferred
 * @param amount Amount of token to be transferred
 * @param recipientChain Destination chain
 * @param recipientAddress Destination wallet address
 * @param relayerFee Relayer fee
 * @param payload Optional payload
 * @returns Transfer messages to be sent on chain
 */
export async function transferFromInjective(
  walletAddress: string,
  tokenBridgeAddress: string,
  tokenAddress: string,
  amount: string,
  recipientChain: ChainId | ChainName,
  recipientAddress: Uint8Array,
  relayerFee: string = "0",
  payload: Uint8Array | null = null
) {
  const recipientChainId = coalesceChainId(recipientChain);
  const nonce = Math.round(Math.random() * 100000);
  const isNativeAsset = isNativeDenomInjective(tokenAddress);
  const mk_action: string = payload
    ? "initiate_transfer_with_payload"
    : "initiate_transfer";
  const mk_initiate_transfer = (info: object) =>
    payload
      ? {
          asset: {
            amount,
            info,
          },
          recipient_chain: recipientChainId,
          recipient: Buffer.from(recipientAddress).toString("base64"),
          fee: relayerFee,
          nonce,
          payload,
        }
      : {
          asset: {
            amount,
            info,
          },
          recipient_chain: recipientChainId,
          recipient: Buffer.from(recipientAddress).toString("base64"),
          fee: relayerFee,
          nonce,
        };
  return isNativeAsset
    ? [
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          msg: {},
          action: "deposit_tokens",
          funds: { denom: tokenAddress, amount },
        }),
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          msg: mk_initiate_transfer({ native_token: { denom: tokenAddress } }),
          action: mk_action,
        }),
      ]
    : [
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenAddress,
          sender: walletAddress,
          msg: {
            spender: tokenBridgeAddress,
            amount,
            expires: {
              never: {},
            },
          },
          action: "increase_allowance",
        }),
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          msg: mk_initiate_transfer({ token: { contract_addr: tokenAddress } }),
          action: mk_action,
        }),
      ];
}

export function transferFromXpla(
  walletAddress: string,
  tokenBridgeAddress: string,
  tokenAddress: string,
  amount: string,
  recipientChain: ChainId | ChainName,
  recipientAddress: Uint8Array,
  relayerFee: string = "0",
  payload: Uint8Array | null = null
): XplaMsgExecuteContract[] {
  const recipientChainId = coalesceChainId(recipientChain);
  const nonce = Math.round(Math.random() * 100000);
  const isNativeAsset = isNativeDenomXpla(tokenAddress);
  const createInitiateTransfer = (info: object) =>
    payload
      ? {
          initiate_transfer_with_payload: {
            asset: {
              amount,
              info,
            },
            recipient_chain: recipientChainId,
            recipient: Buffer.from(recipientAddress).toString("base64"),
            fee: relayerFee,
            nonce,
            payload,
          },
        }
      : {
          initiate_transfer: {
            asset: {
              amount,
              info,
            },
            recipient_chain: recipientChainId,
            recipient: Buffer.from(recipientAddress).toString("base64"),
            fee: relayerFee,
            nonce,
          },
        };
  return isNativeAsset
    ? [
        new XplaMsgExecuteContract(
          walletAddress,
          tokenBridgeAddress,
          {
            deposit_tokens: {},
          },
          { [tokenAddress]: amount }
        ),
        new XplaMsgExecuteContract(
          walletAddress,
          tokenBridgeAddress,
          createInitiateTransfer({
            native_token: {
              denom: tokenAddress,
            },
          }),
          {}
        ),
      ]
    : [
        new XplaMsgExecuteContract(
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
        new XplaMsgExecuteContract(
          walletAddress,
          tokenBridgeAddress,
          createInitiateTransfer({
            token: {
              contract_addr: tokenAddress,
            },
          }),
          {}
        ),
      ];
}

export async function transferNativeSol(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  tokenBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  amount: bigint,
  targetAddress: Uint8Array | Buffer,
  targetChain: ChainId | ChainName,
  relayerFee: bigint = BigInt(0),
  payload: Uint8Array | Buffer | null = null,
  commitment?: Commitment
) {
  const rentBalance = await getMinimumBalanceForRentExemptMint(connection);
  const mintPublicKey = NATIVE_MINT;
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
  const initAccountIx = await createInitializeAccountInstruction(
    ancillaryKeypair.publicKey,
    mintPublicKey,
    payerPublicKey
  );

  //Normal approve & transfer instructions, except that the wSOL is sent from the ancillary account.
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await createBridgeFeeTransferInstruction(
    connection,
    bridgeAddress,
    payerAddress
  );
  const approvalIx = createApproveAuthoritySignerInstruction(
    tokenBridgeAddress,
    ancillaryKeypair.publicKey,
    payerPublicKey,
    amount
  );

  const message = Keypair.generate();
  const tokenBridgeTransferIx =
    payload !== null
      ? createTransferNativeWithPayloadInstruction(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          message.publicKey,
          ancillaryKeypair.publicKey,
          NATIVE_MINT,
          nonce,
          amount,
          Buffer.from(targetAddress),
          coalesceChainId(targetChain),
          payload
        )
      : createTransferNativeInstruction(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          message.publicKey,
          ancillaryKeypair.publicKey,
          NATIVE_MINT,
          nonce,
          amount,
          relayerFee,
          Buffer.from(targetAddress),
          coalesceChainId(targetChain)
        );

  //Close the ancillary account for cleanup. Payer address receives any remaining funds
  const closeAccountIx = createCloseAccountInstruction(
    ancillaryKeypair.publicKey, //account to close
    payerPublicKey, //Remaining funds destination
    payerPublicKey //authority
  );

  const { blockhash } = await connection.getLatestBlockhash(commitment);
  const transaction = new SolanaTransaction();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = payerPublicKey;
  transaction.add(
    createAncillaryAccountIx,
    initialBalanceTransferIx,
    initAccountIx,
    transferIx,
    approvalIx,
    tokenBridgeTransferIx,
    closeAccountIx
  );
  transaction.partialSign(message, ancillaryKeypair);
  return transaction;
}

export async function transferFromSolana(
  connection: Connection,
  bridgeAddress: PublicKeyInitData,
  tokenBridgeAddress: PublicKeyInitData,
  payerAddress: PublicKeyInitData,
  fromAddress: PublicKeyInitData,
  mintAddress: PublicKeyInitData,
  amount: bigint,
  targetAddress: Uint8Array | Buffer,
  targetChain: ChainId | ChainName,
  originAddress?: Uint8Array | Buffer,
  originChain?: ChainId | ChainName,
  fromOwnerAddress?: PublicKeyInitData,
  relayerFee: bigint = BigInt(0),
  payload: Uint8Array | Buffer | null = null,
  commitment?: Commitment
) {
  const originChainId: ChainId | undefined = originChain
    ? coalesceChainId(originChain)
    : undefined;
  if (fromOwnerAddress === undefined) {
    fromOwnerAddress = payerAddress;
  }
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await createBridgeFeeTransferInstruction(
    connection,
    bridgeAddress,
    payerAddress
  );
  const approvalIx = createApproveAuthoritySignerInstruction(
    tokenBridgeAddress,
    fromAddress,
    fromOwnerAddress,
    amount
  );
  const message = Keypair.generate();
  const isSolanaNative =
    originChainId === undefined || originChainId === CHAIN_ID_SOLANA;
  if (!isSolanaNative && !originAddress) {
    return Promise.reject(
      "originAddress is required when specifying originChain"
    );
  }
  const tokenBridgeTransferIx = isSolanaNative
    ? payload !== null
      ? createTransferNativeWithPayloadInstruction(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          message.publicKey,
          fromAddress,
          mintAddress,
          nonce,
          amount,
          targetAddress,
          coalesceChainId(targetChain),
          payload
        )
      : createTransferNativeInstruction(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          message.publicKey,
          fromAddress,
          mintAddress,
          nonce,
          amount,
          relayerFee,
          targetAddress,
          coalesceChainId(targetChain)
        )
    : payload !== null
    ? createTransferWrappedWithPayloadInstruction(
        tokenBridgeAddress,
        bridgeAddress,
        payerAddress,
        message.publicKey,
        fromAddress,
        fromOwnerAddress,
        originChainId!,
        originAddress!,
        nonce,
        amount,
        targetAddress,
        coalesceChainId(targetChain),
        payload
      )
    : createTransferWrappedInstruction(
        tokenBridgeAddress,
        bridgeAddress,
        payerAddress,
        message.publicKey,
        fromAddress,
        fromOwnerAddress,
        originChainId!,
        originAddress!,
        nonce,
        amount,
        relayerFee,
        targetAddress,
        coalesceChainId(targetChain)
      );
  const transaction = new SolanaTransaction().add(
    transferIx,
    approvalIx,
    tokenBridgeTransferIx
  );
  const { blockhash } = await connection.getLatestBlockhash(commitment);
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  transaction.partialSign(message);
  return transaction;
}

/**
 * Transfers an asset from Algorand to a receiver on another chain
 * @param client AlgodV2 client
 * @param tokenBridgeId Application ID of the token bridge
 * @param bridgeId Application ID of the core bridge
 * @param sender Sending account
 * @param assetId Asset index
 * @param qty Quantity to transfer
 * @param receiver Receiving account
 * @param chain Reeiving chain
 * @param fee Transfer fee
 * @param payload payload for payload3 transfers
 * @returns Sequence number of confirmation
 */
export async function transferFromAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  bridgeId: bigint,
  senderAddr: string,
  assetId: bigint,
  qty: bigint,
  receiver: string,
  chain: ChainId | ChainName,
  fee: bigint,
  payload: Uint8Array | null = null
): Promise<TransactionSignerPair[]> {
  const recipientChainId = coalesceChainId(chain);
  const tokenAddr: string = getApplicationAddress(tokenBridgeId);
  const applAddr: string = getEmitterAddressAlgorand(tokenBridgeId);
  const txs: TransactionSignerPair[] = [];
  // "transferAsset"
  const { addr: emitterAddr, txs: emitterOptInTxs } = await optin(
    client,
    senderAddr,
    bridgeId,
    BigInt(0),
    applAddr
  );
  txs.push(...emitterOptInTxs);
  let creator;
  let creatorAcctInfo: any;
  let wormhole: boolean = false;
  if (assetId !== BigInt(0)) {
    const assetInfo: Record<string, any> = await client
      .getAssetByID(safeBigIntToNumber(assetId))
      .do();
    creator = assetInfo["params"]["creator"];
    creatorAcctInfo = await client.accountInformation(creator).do();
    const authAddr: string = creatorAcctInfo["auth-addr"];
    if (authAddr === tokenAddr) {
      wormhole = true;
    }
  }

  const params: SuggestedParams = await client.getTransactionParams().do();
  const msgFee: bigint = await getMessageFee(client, bridgeId);
  if (msgFee > 0) {
    const payTxn: AlgorandTransaction =
      makePaymentTxnWithSuggestedParamsFromObject({
        from: senderAddr,
        suggestedParams: params,
        to: getApplicationAddress(tokenBridgeId),
        amount: msgFee,
      });
    txs.push({ tx: payTxn, signer: null });
  }
  if (!wormhole) {
    const bNat = Buffer.from("native", "binary").toString("hex");
    // "creator"
    const result = await optin(
      client,
      senderAddr,
      tokenBridgeId,
      assetId,
      bNat
    );
    creator = result.addr;
    txs.push(...result.txs);
  }
  if (
    assetId !== BigInt(0) &&
    !(await assetOptinCheck(client, assetId, creator))
  ) {
    // Looks like we need to optin
    const payTxn: AlgorandTransaction =
      makePaymentTxnWithSuggestedParamsFromObject({
        from: senderAddr,
        to: creator,
        amount: 100000,
        suggestedParams: params,
      });
    txs.push({ tx: payTxn, signer: null });
    // The tokenid app needs to do the optin since it has signature authority
    const bOptin: Uint8Array = textToUint8Array("optin");
    let txn = makeApplicationCallTxnFromObject({
      from: senderAddr,
      appIndex: safeBigIntToNumber(tokenBridgeId),
      onComplete: OnApplicationComplete.NoOpOC,
      appArgs: [bOptin, bigIntToBytes(assetId, 8)],
      foreignAssets: [safeBigIntToNumber(assetId)],
      accounts: [creator],
      suggestedParams: params,
    });
    txn.fee *= 2;
    txs.push({ tx: txn, signer: null });
  }
  const t = makeApplicationCallTxnFromObject({
    from: senderAddr,
    appIndex: safeBigIntToNumber(tokenBridgeId),
    onComplete: OnApplicationComplete.NoOpOC,
    appArgs: [textToUint8Array("nop")],
    suggestedParams: params,
  });
  txs.push({ tx: t, signer: null });

  let accounts: string[] = [];
  if (assetId === BigInt(0)) {
    const t = makePaymentTxnWithSuggestedParamsFromObject({
      from: senderAddr,
      to: creator,
      amount: qty,
      suggestedParams: params,
    });
    txs.push({ tx: t, signer: null });
    accounts = [emitterAddr, creator, creator];
  } else {
    const t = makeAssetTransferTxnWithSuggestedParamsFromObject({
      from: senderAddr,
      to: creator,
      suggestedParams: params,
      amount: qty,
      assetIndex: safeBigIntToNumber(assetId),
    });
    txs.push({ tx: t, signer: null });
    accounts = [emitterAddr, creator, creatorAcctInfo["address"]];
  }
  let args = [
    textToUint8Array("sendTransfer"),
    bigIntToBytes(assetId, 8),
    bigIntToBytes(qty, 8),
    hexToUint8Array(receiver),
    bigIntToBytes(recipientChainId, 8),
    bigIntToBytes(fee, 8),
  ];
  if (payload !== null) {
    args.push(payload);
  }
  let acTxn = makeApplicationCallTxnFromObject({
    from: senderAddr,
    appIndex: safeBigIntToNumber(tokenBridgeId),
    onComplete: OnApplicationComplete.NoOpOC,
    appArgs: args,
    foreignApps: [safeBigIntToNumber(bridgeId)],
    foreignAssets: [safeBigIntToNumber(assetId)],
    accounts: accounts,
    suggestedParams: params,
  });
  acTxn.fee *= 2;
  txs.push({ tx: acTxn, signer: null });
  return txs;
}

export async function transferTokenFromNear(
  provider: Provider,
  account: string,
  coreBridge: string,
  tokenBridge: string,
  assetId: string,
  qty: bigint,
  receiver: Uint8Array,
  chain: ChainId | ChainName,
  fee: bigint,
  payload: string = ""
): Promise<FunctionCallOptions[]> {
  const isWrapped = getIsWrappedAssetNear(tokenBridge, assetId);

  const messageFee = await callFunctionNear(
    provider,
    coreBridge,
    "message_fee",
    {}
  );

  chain = coalesceChainId(chain);

  if (isWrapped) {
    return [
      {
        contractId: tokenBridge,
        methodName: "send_transfer_wormhole_token",
        args: {
          token: assetId,
          amount: qty.toString(10),
          receiver: uint8ArrayToHex(receiver),
          chain,
          fee: fee.toString(10),
          payload: payload,
          message_fee: messageFee,
        },
        attachedDeposit: new BN(messageFee + 1),
        gas: new BN("100000000000000"),
      },
    ];
  } else {
    const options: FunctionCallOptions[] = [];
    const bal = await callFunctionNear(
      provider,
      assetId,
      "storage_balance_of",
      {
        account_id: tokenBridge,
      }
    );
    if (bal === null) {
      // Looks like we have to stake some storage for this asset
      // for the token bridge...
      options.push({
        contractId: assetId,
        methodName: "storage_deposit",
        args: { account_id: tokenBridge, registration_only: true },
        gas: new BN("100000000000000"),
        attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
      });
    }

    if (messageFee > 0) {
      const bank = await callFunctionNear(
        provider,
        tokenBridge,
        "bank_balance",
        {
          acct: account,
        }
      );

      if (!bank[0]) {
        options.push({
          contractId: tokenBridge,
          methodName: "register_bank",
          args: {},
          gas: new BN("100000000000000"),
          attachedDeposit: new BN("2000000000000000000000"), // 0.002 NEAR
        });
      }

      if (bank[1] < messageFee) {
        options.push({
          contractId: tokenBridge,
          methodName: "fill_bank",
          args: {},
          gas: new BN("100000000000000"),
          attachedDeposit: new BN(messageFee),
        });
      }
    }

    options.push({
      contractId: assetId,
      methodName: "ft_transfer_call",
      args: {
        receiver_id: tokenBridge,
        amount: qty.toString(10),
        msg: JSON.stringify({
          receiver: uint8ArrayToHex(receiver),
          chain,
          fee: fee.toString(10),
          payload: payload,
          message_fee: messageFee,
        }),
      },
      attachedDeposit: new BN(1),
      gas: new BN("100000000000000"),
    });

    return options;
  }
}

export async function transferNearFromNear(
  provider: Provider,
  coreBridge: string,
  tokenBridge: string,
  qty: bigint,
  receiver: Uint8Array,
  chain: ChainId | ChainName,
  fee: bigint,
  payload: string = ""
): Promise<FunctionCallOptions> {
  const messageFee = await callFunctionNear(
    provider,
    coreBridge,
    "message_fee",
    {}
  );
  return {
    contractId: tokenBridge,
    methodName: "send_transfer_near",
    args: {
      receiver: uint8ArrayToHex(receiver),
      chain: coalesceChainId(chain),
      fee: fee.toString(10),
      payload: payload,
      message_fee: messageFee,
    },
    attachedDeposit: new BN(qty.toString(10)).add(new BN(messageFee)),
    gas: new BN("100000000000000"),
  };
}

/**
 * Transfer an asset on Aptos to another chain.
 * @param tokenBridgeAddress Address of token bridge
 * @param fullyQualifiedType Full qualified type of asset to transfer
 * @param amount Amount to send to recipient
 * @param recipientChain Target chain
 * @param recipient Recipient's address on target chain
 * @param relayerFee Fee to pay relayer
 * @param payload Payload3 data, leave undefined for basic token transfers
 * @returns Transaction payload
 */
export function transferFromAptos(
  tokenBridgeAddress: string,
  fullyQualifiedType: string,
  amount: string,
  recipientChain: ChainId | ChainName,
  recipient: Uint8Array,
  relayerFee: string = "0",
  payload: string = ""
): Types.EntryFunctionPayload {
  if (payload) {
    // Currently unsupported
    return transferTokensWithPayload(
      tokenBridgeAddress,
      fullyQualifiedType,
      amount,
      recipientChain,
      recipient,
      relayerFee,
      createNonce().readUInt32LE(0),
      payload
    );
  }

  return transferTokensAptos(
    tokenBridgeAddress,
    fullyQualifiedType,
    amount,
    recipientChain,
    recipient,
    relayerFee,
    createNonce().readUInt32LE(0)
  );
}
