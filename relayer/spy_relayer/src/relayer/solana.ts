import {
  CHAIN_ID_SOLANA,
  getForeignAssetSolana,
  getIsTransferCompletedSolana,
  hexToNativeString,
  hexToUint8Array,
  importCoreWasm,
  parseTransferPayload,
  postVaaSolanaWithRetry,
  redeemOnSolana,
} from "@certusone/wormhole-sdk";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { ChainConfigInfo } from "../configureEnv";
import { getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";

const MAX_VAA_UPLOAD_RETRIES_SOLANA = 5;

export async function relaySolana(
  chainConfigInfo: ChainConfigInfo,
  signedVAAString: string,
  checkOnly: boolean,
  walletPrivateKey: Uint8Array,
  relayLogger: ScopedLogger,
  metrics: PromHelper
) {
  const logger = getScopedLogger(["solana"], relayLogger);
  //TODO native transfer & create associated token account
  //TODO close connection
  const signedVaaArray = hexToUint8Array(signedVAAString);
  const signedVaaBuffer = Buffer.from(signedVaaArray);
  const connection = new Connection(chainConfigInfo.nodeUrl, "confirmed");

  if (!chainConfigInfo.bridgeAddress) {
    // This should never be the case, as enforced by createSolanaChainConfig
    return { redeemed: false, result: null };
  }

  const keypair = Keypair.fromSecretKey(walletPrivateKey);
  const payerAddress = keypair.publicKey.toString();

  logger.info(
    "publicKey: %s, bridgeAddress: %s, tokenBridgeAddress: %s",
    payerAddress,
    chainConfigInfo.bridgeAddress,
    chainConfigInfo.tokenBridgeAddress
  );
  logger.debug("Checking to see if vaa has already been redeemed.");

  const alreadyRedeemed = await getIsTransferCompletedSolana(
    chainConfigInfo.tokenBridgeAddress,
    signedVaaArray,
    connection
  );

  if (alreadyRedeemed) {
    logger.info("VAA has already been redeemed!");
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }

  // determine fee destination address - an associated token account
  const { parse_vaa } = await importCoreWasm();
  const parsedVAA = parse_vaa(signedVaaArray);
  const payloadBuffer = Buffer.from(parsedVAA.payload);
  const transferPayload = parseTransferPayload(payloadBuffer);
  logger.debug("Calculating the fee destination address");
  const solanaMintAddress =
    transferPayload.originChain === CHAIN_ID_SOLANA
      ? hexToNativeString(transferPayload.originAddress, CHAIN_ID_SOLANA)
      : await getForeignAssetSolana(
          connection,
          chainConfigInfo.tokenBridgeAddress,
          transferPayload.originChain,
          hexToUint8Array(transferPayload.originAddress)
        );
  if (!solanaMintAddress) {
    throw new Error(
      `Unable to determine mint for origin chain: ${
        transferPayload.originChain
      }, address: ${transferPayload.originAddress} (${hexToNativeString(
        transferPayload.originAddress,
        transferPayload.originChain
      )})`
    );
  }
  const solanaMintKey = new PublicKey(solanaMintAddress);
  const feeRecipientAddress = await Token.getAssociatedTokenAddress(
    ASSOCIATED_TOKEN_PROGRAM_ID,
    TOKEN_PROGRAM_ID,
    solanaMintKey,
    keypair.publicKey
  );
  // create the associated token account if it doesn't exist
  const associatedAddressInfo = await connection.getAccountInfo(
    feeRecipientAddress
  );
  if (!associatedAddressInfo) {
    logger.debug(
      "Fee destination address %s for wallet %s, mint %s does not exist, creating it.",
      feeRecipientAddress.toString(),
      keypair.publicKey,
      solanaMintAddress
    );
    const transaction = new Transaction().add(
      await Token.createAssociatedTokenAccountInstruction(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        solanaMintKey,
        feeRecipientAddress,
        keypair.publicKey, // owner
        keypair.publicKey // payer
      )
    );
    const { blockhash } = await connection.getRecentBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = keypair.publicKey;
    // sign, send, and confirm transaction
    transaction.partialSign(keypair);
    const txid = await connection.sendRawTransaction(transaction.serialize());
    await connection.confirmTransaction(txid);
  }

  logger.debug("Posting the vaa.");
  await postVaaSolanaWithRetry(
    connection,
    async (transaction) => {
      transaction.partialSign(keypair);
      return transaction;
    },
    chainConfigInfo.bridgeAddress,
    payerAddress,
    signedVaaBuffer,
    MAX_VAA_UPLOAD_RETRIES_SOLANA
  );

  logger.debug("Redeeming.");
  const unsignedTransaction = await redeemOnSolana(
    connection,
    chainConfigInfo.bridgeAddress,
    chainConfigInfo.tokenBridgeAddress,
    payerAddress,
    signedVaaArray,
    feeRecipientAddress.toString()
  );

  logger.debug("Sending.");
  unsignedTransaction.partialSign(keypair);
  const txid = await connection.sendRawTransaction(
    unsignedTransaction.serialize()
  );
  await connection.confirmTransaction(txid);

  logger.debug("Checking to see if the transaction is complete.");
  const success = await getIsTransferCompletedSolana(
    chainConfigInfo.tokenBridgeAddress,
    signedVaaArray,
    connection
  );

  logger.info("success: %s, tx hash: %s", success, txid);
  metrics.incSuccesses(chainConfigInfo.chainId);
  return { redeemed: success, result: txid };
}
