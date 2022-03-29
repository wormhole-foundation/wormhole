import {
  CHAIN_ID_POLYGON,
  getIsTransferCompletedEth,
  hexToUint8Array,
  redeemOnEth,
  redeemOnEthNative,
} from "@certusone/wormhole-sdk";
import { Signer } from "@ethersproject/abstract-signer";
import { ethers } from "ethers";
import { ChainConfigInfo } from "../configureEnv";
import { getScopedLogger, ScopedLogger } from "../helpers/logHelper";

export function newProvider(
  url: string,
  batch: boolean = false
): ethers.providers.JsonRpcProvider | ethers.providers.JsonRpcBatchProvider {
  // only support http(s), not ws(s) as the websocket constructor can blow up the entire process
  // it uses a nasty setTimeout(()=>{},0) so we are unable to cleanly catch its errors
  if (url.startsWith("http")) {
    if (batch) {
      return new ethers.providers.JsonRpcBatchProvider(url);
    }
    return new ethers.providers.JsonRpcProvider(url);
  }
  throw new Error("url does not start with http/https!");
}

export async function relayEVM(
  chainConfigInfo: ChainConfigInfo,
  signedVAA: string,
  unwrapNative: boolean,
  checkOnly: boolean,
  walletPrivateKey: string,
  relayLogger: ScopedLogger
) {
  const logger = getScopedLogger(
    ["evm", chainConfigInfo.chainName],
    relayLogger
  );
  const signedVaaArray = hexToUint8Array(signedVAA);
  let provider = newProvider(chainConfigInfo.nodeUrl);
  const signer: Signer = new ethers.Wallet(walletPrivateKey, provider);

  if (unwrapNative) {
    logger.info(
      "Will redeem and unwrap using pubkey: %s",
      await signer.getAddress()
    );
  } else {
    logger.info("Will redeem using pubkey: %s", await signer.getAddress());
  }

  logger.debug("Checking to see if vaa has already been redeemed.");
  const alreadyRedeemed = await getIsTransferCompletedEth(
    chainConfigInfo.tokenBridgeAddress,
    provider,
    signedVaaArray
  );

  if (alreadyRedeemed) {
    logger.info("VAA has already been redeemed!");
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }

  logger.debug("Redeeming.");
  // look, there's something janky with Polygon + ethers + EIP-1559
  let overrides;
  if (chainConfigInfo.chainId === CHAIN_ID_POLYGON) {
    let feeData = await provider.getFeeData();
    overrides = {
      maxFeePerGas: feeData.maxFeePerGas?.mul(50) || undefined,
      maxPriorityFeePerGas: feeData.maxPriorityFeePerGas?.mul(50) || undefined,
    };
  }
  const receipt = unwrapNative
    ? await redeemOnEthNative(
        chainConfigInfo.tokenBridgeAddress,
        signer,
        signedVaaArray,
        overrides
      )
    : await redeemOnEth(
        chainConfigInfo.tokenBridgeAddress,
        signer,
        signedVaaArray,
        overrides
      );

  logger.debug("Checking to see if the transaction is complete.");

  const success = await getIsTransferCompletedEth(
    chainConfigInfo.tokenBridgeAddress,
    provider,
    signedVaaArray
  );

  if (provider instanceof ethers.providers.WebSocketProvider) {
    await provider.destroy();
  }

  logger.info("success: %s tx hash: %s", success, receipt.transactionHash);
  return { redeemed: success, result: receipt };
}
