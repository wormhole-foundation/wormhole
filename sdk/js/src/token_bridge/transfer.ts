import {
  JsonRpcProvider,
  SUI_CLOCK_OBJECT_ID,
  SUI_TYPE_ARG,
  TransactionBlock,
} from "@mysten/sui.js";
import {
  ACCOUNT_SIZE,
  NATIVE_MINT,
  TOKEN_PROGRAM_ID,
  createCloseAccountInstruction,
  createInitializeAccountInstruction,
  getMinimumBalanceForRentExemptAccount,
} from "@solana/spl-token";
import {
  Commitment,
  Connection,
  Keypair,
  PublicKey,
  PublicKeyInitData,
  Transaction as SolanaTransaction,
  SystemProgram,
} from "@solana/web3.js";
import { MsgExecuteContract } from "@terra-money/terra.js";
import { MsgExecuteContract as XplaMsgExecuteContract } from "@xpla/xpla.js";
import {
  Algodv2,
  Transaction as AlgorandTransaction,
  OnApplicationComplete,
  SuggestedParams,
  bigIntToBytes,
  getApplicationAddress,
  makeApplicationCallTxnFromObject,
  makeAssetTransferTxnWithSuggestedParamsFromObject,
  makePaymentTxnWithSuggestedParamsFromObject,
} from "algosdk";
import { Types } from "aptos";
import BN from "bn.js";
import { Overrides, PayableOverrides, ethers } from "ethers";
import { FunctionCallOptions } from "near-api-js/lib/account";
import { Provider } from "near-api-js/lib/providers";
import { getIsWrappedAssetNear } from "..";
import {
  TransactionSignerPair,
  assetOptinCheck,
  getMessageFee,
  optin,
} from "../algorand";
import {
  transferTokens as transferTokensAptos,
  transferTokensWithPayload,
} from "../aptos";
import { getEmitterAddressAlgorand } from "../bridge";
import { isNativeDenomXpla } from "../cosmwasm";
import {
  Bridge__factory,
  TokenImplementation__factory,
} from "../ethers-contracts";
import {
  createApproveAuthoritySignerInstruction,
  createTransferNativeInstruction,
  createTransferNativeWithPayloadInstruction,
  createTransferWrappedInstruction,
  createTransferWrappedWithPayloadInstruction,
} from "../solana/tokenBridge";
import { getOldestEmitterCapObjectId, getPackageId, isSameType } from "../sui";
import { SuiCoinObject } from "../sui/types";
import { isNativeDenom } from "../terra";
import {
  CHAIN_ID_SOLANA,
  ChainId,
  ChainName,
  callFunctionNear,
  coalesceChainId,
  createNonce,
  hexToUint8Array,
  safeBigIntToNumber,
  textToUint8Array,
  uint8ArrayToHex,
} from "../utils";

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
  const rentBalance = await getMinimumBalanceForRentExemptAccount(
    connection,
    commitment
  );
  const payerPublicKey = new PublicKey(payerAddress);
  const ancillaryKeypair = Keypair.generate();

  //This will create a temporary account where the wSOL will be created.
  const createAncillaryAccountIx = SystemProgram.createAccount({
    fromPubkey: payerPublicKey,
    newAccountPubkey: ancillaryKeypair.publicKey,
    lamports: rentBalance, //spl token accounts need rent exemption
    space: ACCOUNT_SIZE,
    programId: TOKEN_PROGRAM_ID,
  });

  //Send in the amount of SOL which we want converted to wSOL
  const initialBalanceTransferIx = SystemProgram.transfer({
    fromPubkey: payerPublicKey,
    lamports: amount,
    toPubkey: ancillaryKeypair.publicKey,
  });
  //Initialize the account as a WSOL account, with the original payerAddress as owner
  const initAccountIx = createInitializeAccountInstruction(
    ancillaryKeypair.publicKey,
    NATIVE_MINT,
    payerPublicKey
  );

  //Normal approve & transfer instructions, except that the wSOL is sent from the ancillary account.
  const approvalIx = createApproveAuthoritySignerInstruction(
    tokenBridgeAddress,
    ancillaryKeypair.publicKey,
    payerPublicKey,
    amount
  );

  const message = Keypair.generate();
  const nonce = createNonce().readUInt32LE(0);
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
 * @param payload Payload3 data, leave null for basic token transfers
 * @returns Transaction payload
 */
export function transferFromAptos(
  tokenBridgeAddress: string,
  fullyQualifiedType: string,
  amount: string,
  recipientChain: ChainId | ChainName,
  recipient: Uint8Array,
  relayerFee: string = "0",
  payload: Uint8Array | null = null
): Types.EntryFunctionPayload {
  if (payload) {
    // Currently unsupported
    return transferTokensWithPayload(
      tokenBridgeAddress,
      fullyQualifiedType,
      amount,
      recipientChain,
      recipient,
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

/**
 * Transfer an asset from Sui to another chain.
 */
export async function transferFromSui(
  provider: JsonRpcProvider,
  coreBridgeStateObjectId: string,
  tokenBridgeStateObjectId: string,
  coins: SuiCoinObject[],
  coinType: string,
  amount: bigint,
  recipientChain: ChainId | ChainName,
  recipient: Uint8Array,
  feeAmount: bigint = BigInt(0),
  relayerFee: bigint = BigInt(0),
  payload: Uint8Array | null = null,
  coreBridgePackageId?: string,
  tokenBridgePackageId?: string,
  senderAddress?: string
): Promise<TransactionBlock> {
  const [primaryCoin, ...mergeCoins] = coins.filter((coin) =>
    isSameType(coin.coinType, coinType)
  );
  if (primaryCoin === undefined) {
    throw new Error(
      `Coins array doesn't contain any coins of type ${coinType}`
    );
  }

  [coreBridgePackageId, tokenBridgePackageId] = await Promise.all([
    coreBridgePackageId
      ? Promise.resolve(coreBridgePackageId)
      : getPackageId(provider, coreBridgeStateObjectId),
    tokenBridgePackageId
      ? Promise.resolve(tokenBridgePackageId)
      : getPackageId(provider, tokenBridgeStateObjectId),
  ]);
  const tx = new TransactionBlock();
  const [transferCoin] = (() => {
    if (coinType === SUI_TYPE_ARG) {
      return tx.splitCoins(tx.gas, [tx.pure(amount)]);
    } else {
      const primaryCoinInput = tx.object(primaryCoin.coinObjectId);
      if (mergeCoins.length) {
        tx.mergeCoins(
          primaryCoinInput,
          mergeCoins.map((coin) => tx.object(coin.coinObjectId))
        );
      }

      return tx.splitCoins(primaryCoinInput, [tx.pure(amount)]);
    }
  })();
  const [feeCoin] = tx.splitCoins(tx.gas, [tx.pure(feeAmount)]);
  const [assetInfo] = tx.moveCall({
    target: `${tokenBridgePackageId}::state::verified_asset`,
    arguments: [tx.object(tokenBridgeStateObjectId)],
    typeArguments: [coinType],
  });
  if (payload === null) {
    const [transferTicket, dust] = tx.moveCall({
      target: `${tokenBridgePackageId}::transfer_tokens::prepare_transfer`,
      arguments: [
        assetInfo,
        transferCoin,
        tx.pure(coalesceChainId(recipientChain)),
        tx.pure([...recipient]),
        tx.pure(relayerFee),
        tx.pure(createNonce().readUInt32LE()),
      ],
      typeArguments: [coinType],
    });
    tx.moveCall({
      target: `${tokenBridgePackageId}::coin_utils::return_nonzero`,
      arguments: [dust],
      typeArguments: [coinType],
    });
    const [messageTicket] = tx.moveCall({
      target: `${tokenBridgePackageId}::transfer_tokens::transfer_tokens`,
      arguments: [tx.object(tokenBridgeStateObjectId), transferTicket],
      typeArguments: [coinType],
    });
    tx.moveCall({
      target: `${coreBridgePackageId}::publish_message::publish_message`,
      arguments: [
        tx.object(coreBridgeStateObjectId),
        feeCoin,
        messageTicket,
        tx.object(SUI_CLOCK_OBJECT_ID),
      ],
    });
    return tx;
  } else {
    if (!senderAddress) {
      throw new Error("senderAddress is required for transfer with payload");
    }
    // Get or create a new `EmitterCap`
    let isNewEmitterCap = false;
    const emitterCap = await (async () => {
      const objectId = await getOldestEmitterCapObjectId(
        provider,
        coreBridgePackageId,
        senderAddress
      );
      if (objectId !== null) {
        return tx.object(objectId);
      } else {
        const [emitterCap] = tx.moveCall({
          target: `${coreBridgePackageId}::emitter::new`,
          arguments: [tx.object(coreBridgeStateObjectId)],
        });
        isNewEmitterCap = true;
        return emitterCap;
      }
    })();
    const [transferTicket, dust] = tx.moveCall({
      target: `${tokenBridgePackageId}::transfer_tokens_with_payload::prepare_transfer`,
      arguments: [
        emitterCap,
        assetInfo,
        transferCoin,
        tx.pure(coalesceChainId(recipientChain)),
        tx.pure([...recipient]),
        tx.pure([...payload]),
        tx.pure(createNonce().readUInt32LE()),
      ],
      typeArguments: [coinType],
    });
    tx.moveCall({
      target: `${tokenBridgePackageId}::coin_utils::return_nonzero`,
      arguments: [dust],
      typeArguments: [coinType],
    });
    const [messageTicket] = tx.moveCall({
      target: `${tokenBridgePackageId}::transfer_tokens_with_payload::transfer_tokens_with_payload`,
      arguments: [tx.object(tokenBridgeStateObjectId), transferTicket],
      typeArguments: [coinType],
    });
    tx.moveCall({
      target: `${coreBridgePackageId}::publish_message::publish_message`,
      arguments: [
        tx.object(coreBridgeStateObjectId),
        feeCoin,
        messageTicket,
        tx.object(SUI_CLOCK_OBJECT_ID),
      ],
    });
    if (isNewEmitterCap) {
      tx.transferObjects([emitterCap], tx.pure(senderAddress));
    }
    return tx;
  }
}
