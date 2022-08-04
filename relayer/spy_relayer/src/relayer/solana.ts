import {
  getIsTransferCompletedSolana,
  hexToUint8Array,
} from "@certusone/wormhole-sdk";
import { Connection } from "@solana/web3.js";
import { ChainConfigInfo } from "../configureEnv";
import { getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import {
  relayToSolana,
  relayToSolanaWithFailure,
} from "../xRaydium/scripts/relay";
import * as xRaydiumLib from "../xRaydium/scripts/lib/lib";
import * as whHelpers from '../xRaydium/scripts/lib/wh_helpers'
import * as devnet_ctx from "../xRaydium/scripts/lib/devnet_ctx";
import { chainConfigToEvmProviderAndSigner } from "./evm";
import { providers } from "ethers";
import { _undef } from "../xRaydium/scripts/lib/utilities";

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
  console.log("relaySolana chainConfigInfo: ", chainConfigInfo);
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
  const { transfer, baseVAA } = await whHelpers.parseTransferTokenWithPayload(
    signedVaaArray
  );

  const {signer, provider} = await chainConfigToEvmProviderAndSigner(emitterChainConfigInfo)
  const ctx: xRaydiumLib.Context = devnet_ctx.getDevNetCtx(
    signer, 
    emitterChainConfigInfo.chainId,
    _undef(emitterChainConfigInfo.walletPrivateKey, "expected emitter chain to have wallet private key")[0],
    provider,
  );

  const header = await whHelpers.parseHeaderFromPayload3(transfer.payload3);
  const escrowState = await xRaydiumLib.tryFetchEscrowState(ctx.sol, transfer, header, {
    silent: true,
    retries: 2,
  });
  if (
    alreadyRedeemed &&
    escrowState &&
    (escrowState.escrowStateMarker.kind === "Completed" ||
      escrowState.escrowStateMarker.kind === "Aborted") &&
    escrowState.inputTokens.every((t) => t.hasBeenReturned) &&
    escrowState.outputTokens.every((t) => t.hasBeenReturned)
  ) {
    logger.info("VAA has already been redeemed!");
    return { redeemed: true, result: "already redeemed" };
  }
  if (checkOnly) {
    return { redeemed: false, result: "not redeemed" };
  }

  await relayToSolana(ctx, signedVaaArray, baseVAA, transfer);

  logger.info("\n\n============= Done relaying to solana ============\n\n");

  return { redeemed: true, result: "redeemed" };
}
