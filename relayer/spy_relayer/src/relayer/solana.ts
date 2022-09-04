import {
  getIsTransferCompletedSolana,
  hexToUint8Array,
} from "@certusone/wormhole-sdk";
import { Connection } from "@solana/web3.js";
import { ChainConfigInfo } from "../configureEnv";
import { getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import { chainConfigToEvmProviderAndSigner } from "./evm";
import * as xApp from "../xRaydium/scripts/lib";
import * as relay from "../xRaydium/scripts/relay";

const MAX_VAA_UPLOAD_RETRIES_SOLANA = 5;

export async function relaySolana(
  chainConfigInfo: ChainConfigInfo,
  emitterChainConfigInfo: ChainConfigInfo,
  signedVAAString: string,
  checkOnly: boolean,
  walletPrivateKey: Uint8Array,
  relayLogger: ScopedLogger,
  metrics: PromHelper
) {
  console.log("signedVAAString: ", signedVAAString);
  const logger = getScopedLogger(["solana"], relayLogger);
  console.log("relaySolana chainConfigInfo: ", {chainConfigInfo, so});
  //TODO native transfer & create associated token account
  //TODO close connection
  const signedVaaArray = hexToUint8Array(signedVAAString);
  const connection = new Connection(chainConfigInfo.nodeUrl, "confirmed");
  if (!chainConfigInfo.bridgeAddress) {
    // This should never be the case, as enforced by createSolanaChainConfig
    return { redeemed: false, result: null };
  }

  console.log("==============in relaySolana.ts==============");
  console.log(
    "chainConfigInfo.tokenBridgeAddress: ",
    chainConfigInfo.tokenBridgeAddress
  );
  const alreadyRedeemed = await getIsTransferCompletedSolana(
    chainConfigInfo.tokenBridgeAddress,
    signedVaaArray,
    connection
  );
  //@ts-ignore
  const { transfer, baseVAA } = await xApp.parseTransferTokenWithPayload(
    signedVaaArray
  );

  const { signer, provider } = await chainConfigToEvmProviderAndSigner(
    emitterChainConfigInfo
  );
  const addrs = await xApp.loadAddrs();
  let ctx: xApp.Context;
  if (process.env.ENV_TYPE === "DEV_NET") {
    ctx = xApp.getDevNetCtx(
      signer,
      emitterChainConfigInfo.chainId,
      xApp._undef(
        emitterChainConfigInfo.walletPrivateKey,
        "expected emitter chain to have wallet private key"
      )[0],
      addrs.fuji.XRaydiumBridge,
      provider
    );
  } else {
    ctx = xApp.getAvaxMainnetCtx(addrs.avax.XRaydiumBridge);
  }

  const header = await xApp.parseHeaderFromPayload3(transfer.payload3, true);
  const escrowState = await xApp.tryFetchEscrowState(
    ctx.sol,
    transfer,
    header,
    {
      silent: true,
      retries: 2,
    }
  );
  if (
    alreadyRedeemed &&
    escrowState &&
    (escrowState.marker.kind === "Completed" ||
      escrowState.marker.kind === "Aborted") &&
    escrowState.inputTokens.every((t) => t.returnVaaSequence !== null) &&
    escrowState.outputTokens.every((t) => t.returnVaaSequence !== null)
  ) {
    logger.info("VAA has already been redeemed!");
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }

  await relay.relayToSolana(ctx, signedVaaArray, baseVAA, transfer);

  logger.info("\n\n============= Done relaying to solana ============\n\n");

  return { redeemed: true, result: "redeemed" };
}
