import {
  CHAIN_ID_SOLANA,
  getForeignAssetSolana,
  getIsTransferCompletedSolana,
  hexToNativeString,
  hexToUint8Array,
  importCoreWasm,
  parseTransferPayload,
  postVaaSolana,
  redeemOnSolana,
} from "@certusone/wormhole-sdk";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { ChainConfigInfo } from "../configureEnv";
import { getLogger } from "../helpers/logHelper";

const logger = getLogger();

export async function relaySolana(
  chainConfigInfo: ChainConfigInfo,
  signedVAAString: string,
  checkOnly: boolean,
  walletPrivateKey: Uint8Array
) {
  //TODO native transfer & create associated token account
  //TODO close connection
  const signedVaaArray = hexToUint8Array(signedVAAString);
  const signedVaaBuffer = Buffer.from(signedVaaArray);
  const connection = new Connection(chainConfigInfo.nodeUrl, "confirmed");

  if (!chainConfigInfo.bridgeAddress) {
    return { redeemed: false, result: null };
  }

  //TODO log public key here
  logger.info(
    "relaySolana tokenBridgeAddress: [" +
      chainConfigInfo.tokenBridgeAddress +
      "]"
  );
  logger.info("bridgeAddress: [" + chainConfigInfo.bridgeAddress + "]");
  // logger.info("signedVAAString: [" + signedVAAString + "]");
  // logger.info(" signedVaaArray: %o", signedVaaArray);
  // logger.info(", signedVaaBuffer: %o", signedVaaBuffer);
  // logger.info(", connection: %o", connection);

  logger.debug(
    "relaySolana: checking to see if vaa has already been redeemed."
  );
  const alreadyRedeemed = await getIsTransferCompletedSolana(
    chainConfigInfo.tokenBridgeAddress,
    signedVaaArray,
    connection
  );

  if (alreadyRedeemed) {
    logger.info("relaySolana: vaa has already been redeemed!");
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }

  const keypair = Keypair.fromSecretKey(walletPrivateKey);
  const payerAddress = keypair.publicKey.toString();

  // determine fee destination address - an associated token account
  const { parse_vaa } = await importCoreWasm();
  const parsedVAA = parse_vaa(signedVaaArray);
  const payload = parseTransferPayload(parsedVAA);
  logger.debug("relaySolana: calculating the fee destination address");
  const solanaMintAddress =
    payload.originChain === CHAIN_ID_SOLANA
      ? hexToNativeString(payload.originAddress, CHAIN_ID_SOLANA)
      : await getForeignAssetSolana(
          connection,
          chainConfigInfo.tokenBridgeAddress,
          payload.originChain,
          hexToUint8Array(payload.originAddress)
        );
  if (!solanaMintAddress) {
    throw new Error(
      `Unable to determine mint for origin chain: ${
        payload.originChain
      }, address: ${payload.originAddress} (${hexToNativeString(
        payload.originAddress,
        payload.originChain
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
      "relaySolana: fee destination address %s for wallet %s, mint %s does not exist, creating it.",
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

  logger.debug("relaySolana: posting the vaa.");
  await postVaaSolana(
    connection,
    async (transaction) => {
      transaction.partialSign(keypair);
      return transaction;
    },
    chainConfigInfo.bridgeAddress,
    payerAddress,
    signedVaaBuffer
  );

  logger.debug("relaySolana: redeeming.");
  const unsignedTransaction = await redeemOnSolana(
    connection,
    chainConfigInfo.bridgeAddress,
    chainConfigInfo.tokenBridgeAddress,
    payerAddress,
    signedVaaArray,
    feeRecipientAddress.toString()
  );

  logger.debug("relaySolana: sending.");
  unsignedTransaction.partialSign(keypair);
  const txid = await connection.sendRawTransaction(
    unsignedTransaction.serialize()
  );
  await connection.confirmTransaction(txid);

  logger.debug("relaySolana: checking to see if the transaction is complete.");
  const success = await getIsTransferCompletedSolana(
    chainConfigInfo.tokenBridgeAddress,
    signedVaaArray,
    connection
  );

  logger.info("relaySolana: success: %s, tx hash: %s", success, txid);
  return { redeemed: success, result: txid };
}
