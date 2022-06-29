import { AccountLayout, Token, TOKEN_PROGRAM_ID, u64 } from "@solana/spl-token";
import {
  Connection,
  Keypair,
  PublicKey,
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
import { BigNumber, ethers, Overrides, PayableOverrides } from "ethers";
import { isNativeDenom, isNativeDenomInjective } from "..";
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
import { getBridgeFeeIx, ixFromRust } from "../solana";
import { importTokenWasm } from "../solana/wasm";
import {
  ChainId,
  ChainName,
  CHAIN_ID_SOLANA,
  coalesceChainId,
  createNonce,
  hexToUint8Array,
  textToUint8Array,
  WSOL_ADDRESS,
} from "../utils";
import { safeBigIntToNumber } from "../utils/bigint";

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
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          msg: { deposit_tokens: {} },
          amount: { denom: "inj", amount: amount },
        }),
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          msg: mk_initiate_transfer({ native_token: { denom: tokenAddress } }),
        }),
      ]
    : [
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          msg: {
            increase_allowance: {
              spender: tokenBridgeAddress,
              amount: amount,
              expires: {
                never: {},
              },
            },
          },
        }),
        MsgExecuteContractInjective.fromJSON({
          contractAddress: tokenBridgeAddress,
          sender: walletAddress,
          msg: mk_initiate_transfer({ token: { contract_addr: tokenAddress } }),
        }),
      ];
}

export async function transferNativeSol(
  connection: Connection,
  bridgeAddress: string,
  tokenBridgeAddress: string,
  payerAddress: string,
  amount: BigInt,
  targetAddress: Uint8Array,
  targetChain: ChainId | ChainName,
  relayerFee: BigInt = BigInt(0),
  payload: Uint8Array | null = null
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
  const {
    transfer_native_ix,
    transfer_native_with_payload_ix,
    approval_authority_address,
  } = await importTokenWasm();
  const nonce = createNonce().readUInt32LE(0);
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
    payload !== null
      ? transfer_native_with_payload_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          messageKey.publicKey.toString(),
          ancillaryKeypair.publicKey.toString(),
          WSOL_ADDRESS,
          nonce,
          amount.valueOf(),
          targetAddress,
          coalesceChainId(targetChain),
          payload
        )
      : transfer_native_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          messageKey.publicKey.toString(),
          ancillaryKeypair.publicKey.toString(),
          WSOL_ADDRESS,
          nonce,
          amount.valueOf(),
          relayerFee.valueOf(),
          targetAddress,
          coalesceChainId(targetChain)
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
  const transaction = new SolanaTransaction();
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
  targetChain: ChainId | ChainName,
  originAddress?: Uint8Array,
  originChain?: ChainId | ChainName,
  fromOwnerAddress?: string,
  relayerFee: BigInt = BigInt(0),
  payload: Uint8Array | null = null
) {
  const originChainId: ChainId | undefined = originChain
    ? coalesceChainId(originChain)
    : undefined;
  const nonce = createNonce().readUInt32LE(0);
  const transferIx = await getBridgeFeeIx(
    connection,
    bridgeAddress,
    payerAddress
  );
  const {
    transfer_native_ix,
    transfer_wrapped_ix,
    transfer_native_with_payload_ix,
    transfer_wrapped_with_payload_ix,
    approval_authority_address,
  } = await importTokenWasm();
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
    originChainId === undefined || originChainId === CHAIN_ID_SOLANA;
  if (!isSolanaNative && !originAddress) {
    throw new Error("originAddress is required when specifying originChain");
  }
  const ix = ixFromRust(
    isSolanaNative
      ? payload !== null
        ? transfer_native_with_payload_ix(
            tokenBridgeAddress,
            bridgeAddress,
            payerAddress,
            messageKey.publicKey.toString(),
            fromAddress,
            mintAddress,
            nonce,
            amount.valueOf(),
            targetAddress,
            coalesceChainId(targetChain),
            payload
          )
        : transfer_native_ix(
            tokenBridgeAddress,
            bridgeAddress,
            payerAddress,
            messageKey.publicKey.toString(),
            fromAddress,
            mintAddress,
            nonce,
            amount.valueOf(),
            relayerFee.valueOf(),
            targetAddress,
            coalesceChainId(targetChain)
          )
      : payload !== null
      ? transfer_wrapped_with_payload_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          messageKey.publicKey.toString(),
          fromAddress,
          fromOwnerAddress || payerAddress,
          originChainId as number, // checked by isSolanaNative
          originAddress as Uint8Array, // checked by throw
          nonce,
          amount.valueOf(),
          targetAddress,
          coalesceChainId(targetChain),
          payload
        )
      : transfer_wrapped_ix(
          tokenBridgeAddress,
          bridgeAddress,
          payerAddress,
          messageKey.publicKey.toString(),
          fromAddress,
          fromOwnerAddress || payerAddress,
          originChainId as number, // checked by isSolanaNative
          originAddress as Uint8Array, // checked by throw
          nonce,
          amount.valueOf(),
          relayerFee.valueOf(),
          targetAddress,
          coalesceChainId(targetChain)
        )
  );
  const transaction = new SolanaTransaction().add(transferIx, approvalIx, ix);
  const { blockhash } = await connection.getRecentBlockhash();
  transaction.recentBlockhash = blockhash;
  transaction.feePayer = new PublicKey(payerAddress);
  transaction.partialSign(messageKey);
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
