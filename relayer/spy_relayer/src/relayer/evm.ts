import {
  getIsTransferCompletedEth,
  hexToUint8Array,
  redeemOnEth,
  redeemOnEthNative,
} from "@certusone/wormhole-sdk";
import { Signer } from "@ethersproject/abstract-signer";
import { ethers } from "ethers";
import { ChainConfigInfo } from "../configureEnv";
import { getLogger } from "../helpers/logHelper";

const logger = getLogger();

export function newProvider(
  url: string
): ethers.providers.WebSocketProvider | ethers.providers.JsonRpcProvider {
  if (url.startsWith("ws")) {
    return new ethers.providers.WebSocketProvider(url);
  } else if (url.startsWith("http")) {
    return new ethers.providers.JsonRpcProvider(url);
  }
  throw new Error("url does not start with ws or http!");
}

export async function relayEVM(
  chainConfigInfo: ChainConfigInfo,
  signedVAA: string,
  unwrapNative: boolean,
  checkOnly: boolean,
  walletPrivateKey: string
) {
  const signedVaaArray = hexToUint8Array(signedVAA);
  let provider = newProvider(chainConfigInfo.nodeUrl);
  const signer: Signer = new ethers.Wallet(walletPrivateKey, provider);

  logger.info(
    "relayEVM(" +
      chainConfigInfo.chainName +
      "): " +
      (unwrapNative ? ", will unwrap" : "") +
      ", " +
      "pubkey : " +
      signer.getAddress()
  );

  logger.debug(
    "relayEVM(" +
      chainConfigInfo.chainName +
      "): checking to see if vaa has already been redeemed."
  );
  const alreadyRedeemed = await getIsTransferCompletedEth(
    chainConfigInfo.tokenBridgeAddress,
    provider,
    signedVaaArray
  );

  if (alreadyRedeemed) {
    logger.info(
      "relayEVM(" +
        chainConfigInfo.chainName +
        "): vaa has already been redeemed!"
    );
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }

  logger.debug("relayEVM(" + chainConfigInfo.chainName + "): redeeming.");
  const receipt = unwrapNative
    ? await redeemOnEthNative(
        chainConfigInfo.tokenBridgeAddress,
        signer,
        signedVaaArray
      )
    : await redeemOnEth(
        chainConfigInfo.tokenBridgeAddress,
        signer,
        signedVaaArray
      );

  logger.debug(
    "relayEVM(" +
      chainConfigInfo.chainName +
      "): checking to see if the transaction is complete."
  );

  const success = await getIsTransferCompletedEth(
    chainConfigInfo.tokenBridgeAddress,
    provider,
    signedVaaArray
  );

  if (provider instanceof ethers.providers.WebSocketProvider) {
    await provider.destroy();
  }

  logger.info(
    "relayEVM(" +
      chainConfigInfo.chainName +
      "): success: " +
      success +
      ", receipt: %o",
    receipt
  );
  return { redeemed: success, result: receipt };
}
