import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  transferFromEth,
  transferFromEthNative,
  transferFromSolana,
  transferNativeSol,
} from "@certusone/wormhole-sdk";
import { parseUnits } from "@ethersproject/units";
import { Connection, Keypair } from "@solana/web3.js";
import {
  getBridgeAddressForChain,
  getSignerForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "../consts";

/*
  This function transfers the given token and returns the resultant sequence number, which is then used to retrieve the
  VAA from the guardians.
  */
export async function transferTokens(
  sourceChain: ChainId,
  amount: string,
  targetChain: ChainId,
  sourceAddress: string,
  recipientAddress: string,
  isNativeAsset: boolean,
  assetAddress?: string,
  decimals?: number
): Promise<string> {
  //TODO support native assets,
  //TODO set relayer fee,
  if (sourceChain === CHAIN_ID_SOLANA) {
    return transferSolana(
      amount,
      targetChain,
      sourceAddress,
      recipientAddress,
      isNativeAsset,
      assetAddress,
      decimals
    );
  } else if (sourceChain === CHAIN_ID_TERRA) {
    return transferTerra(
      amount,
      targetChain,
      recipientAddress,
      isNativeAsset,
      assetAddress,
      decimals
    );
  } else {
    return transferEvm(
      sourceChain,
      amount,
      targetChain,
      recipientAddress,
      isNativeAsset,
      assetAddress,
      decimals
    );
  }
}

export async function transferSolana(
  amount: string,
  targetChain: ChainId,
  sourceAddress: string,
  recipientAddress: string,
  isNativeAsset: boolean,
  assetAddress?: string,
  decimals?: number
): Promise<string> {
  if (isNativeAsset) {
    decimals = 9;
  } else if (!assetAddress || !decimals) {
    throw new Error("No token specified for transfer.");
  }
  const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
  const payerAddress = keypair.publicKey.toString();
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const amountParsed = parseUnits(amount, decimals).toBigInt();
  const hexString = nativeToHexString(recipientAddress, targetChain);
  if (!hexString) {
    throw new Error("Invalid recipient");
  }
  const vaaCompatibleAddress = hexToUint8Array(hexString);
  const promise = isNativeAsset
    ? transferNativeSol(
        connection,
        SOL_BRIDGE_ADDRESS,
        SOL_TOKEN_BRIDGE_ADDRESS,
        payerAddress,
        amountParsed,
        vaaCompatibleAddress,
        targetChain
      )
    : transferFromSolana(
        connection,
        SOL_BRIDGE_ADDRESS,
        SOL_TOKEN_BRIDGE_ADDRESS,
        payerAddress, //Actual SOL fee paying address
        sourceAddress, //SPL token account
        assetAddress as string,
        amountParsed,
        vaaCompatibleAddress,
        targetChain
        //TODO support non-wormhole assets here.
        //   originAddress,
        //   originChain
      );
  const transaction = await promise;
  transaction.partialSign(keypair);
  const txid = await connection.sendRawTransaction(transaction.serialize());
  await connection.confirmTransaction(txid);
  const info = await connection.getTransaction(txid);
  if (!info) {
    throw new Error("An error occurred while fetching the transaction info");
  }
  const sequence = parseSequenceFromLogSolana(info);
  return sequence;
}
export async function transferTerra(
  amount: string,
  targetChain: ChainId,
  recipientAddress: string,
  isNativeAsset: boolean,
  assetAddress?: string,
  decimals?: number
): Promise<string> {
  //TODO modify bridge_ui to use in-memory signer
  throw new Error("Unimplemented");
}

export async function transferEvm(
  sourceChain: ChainId,
  amount: string,
  targetChain: ChainId,
  recipientAddress: string,
  isNativeAsset: boolean,
  assetAddress?: string,
  decimals?: number
): Promise<string> {
  if (isNativeAsset) {
    decimals = 18;
  } else if (!assetAddress || !decimals) {
    throw new Error("No token specified for transfer.");
  }
  const amountParsed = parseUnits(amount, decimals);
  const signer = getSignerForChain(sourceChain);
  const hexString = nativeToHexString(recipientAddress, targetChain);
  if (!hexString) {
    throw new Error("Invalid recipient");
  }
  const vaaCompatibleAddress = hexToUint8Array(hexString);
  const receipt = isNativeAsset
    ? await transferFromEthNative(
        getTokenBridgeAddressForChain(sourceChain),
        signer,
        amountParsed,
        targetChain,
        vaaCompatibleAddress
      )
    : await transferFromEth(
        getTokenBridgeAddressForChain(sourceChain),
        signer,
        assetAddress as string,
        amountParsed,
        targetChain,
        vaaCompatibleAddress
      );

  return await parseSequenceFromLogEth(
    receipt,
    getBridgeAddressForChain(sourceChain)
  );
}
