import { importCoreWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";

import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  hexToNativeString,
  hexToUint8Array,
  isEVMChain,
  parseTransferPayload,
} from "@certusone/wormhole-sdk";

import { relayEVM } from "./evm";
import { relaySolana } from "./solana";
import { relayTerra } from "./terra";
import { getRelayerEnvironment } from "../configureEnv";
import { RelayResult, Status } from "../helpers/redisHelper";
import { getLogger, getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";

const logger = getLogger();

function getChainConfigInfo(chainId: ChainId) {
  const env = getRelayerEnvironment();
  return env.supportedChains.find((x) => x.chainId === chainId);
}

export async function relay(
  signedVAA: string,
  checkOnly: boolean,
  walletPrivateKey: any,
  relayLogger: ScopedLogger,
  metrics: PromHelper
): Promise<RelayResult> {
  const logger = getScopedLogger(["relay"], relayLogger);
  const { parse_vaa } = await importCoreWasm();
  const parsedVAA = parse_vaa(hexToUint8Array(signedVAA));
  if (parsedVAA.payload[0] === 1) {
    const transferPayload = parseTransferPayload(
      Buffer.from(parsedVAA.payload)
    );

    const chainConfigInfo = getChainConfigInfo(transferPayload.targetChain);
    if (!chainConfigInfo) {
      logger.error("relay: improper chain ID: " + transferPayload.targetChain);
      return {
        status: Status.FatalError,
        result:
          "Fatal Error: target chain " +
          transferPayload.targetChain +
          " not supported",
      };
    }

    if (isEVMChain(transferPayload.targetChain)) {
      const unwrapNative =
        transferPayload.originChain === transferPayload.targetChain &&
        hexToNativeString(
          transferPayload.originAddress,
          transferPayload.originChain
        )?.toLowerCase() === chainConfigInfo.wrappedAsset?.toLowerCase();
      logger.debug(
        "isEVMChain: originAddress: [" +
          transferPayload.originAddress +
          "], wrappedAsset: [" +
          chainConfigInfo.wrappedAsset +
          "], unwrapNative: " +
          unwrapNative
      );
      let evmResult = await relayEVM(
        chainConfigInfo,
        signedVAA,
        unwrapNative,
        checkOnly,
        walletPrivateKey,
        logger,
        metrics
      );
      return {
        status: evmResult.redeemed ? Status.Completed : Status.Error,
        result: evmResult.result.toString(),
      };
    }

    if (transferPayload.targetChain === CHAIN_ID_SOLANA) {
      let rResult: RelayResult = { status: Status.Error, result: "" };
      const retVal = await relaySolana(
        chainConfigInfo,
        signedVAA,
        checkOnly,
        walletPrivateKey,
        logger,
        metrics
      );
      if (retVal.redeemed) {
        rResult.status = Status.Completed;
      }
      rResult.result = retVal.result;
      return rResult;
    }

    if (transferPayload.targetChain === CHAIN_ID_TERRA) {
      let rResult: RelayResult = { status: Status.Error, result: "" };
      const retVal = await relayTerra(
        chainConfigInfo,
        signedVAA,
        checkOnly,
        walletPrivateKey,
        logger,
        metrics
      );
      if (retVal.redeemed) {
        rResult.status = Status.Completed;
      }
      rResult.result = retVal.result;
      return rResult;
    }

    logger.error(
      "relay: target chain ID: " +
        transferPayload.targetChain +
        " is invalid, this is a program bug!"
    );

    return {
      status: Status.FatalError,
      result:
        "Fatal Error: target chain " +
        transferPayload.targetChain +
        " is invalid, this is a program bug!",
    };
  }
  return { status: Status.FatalError, result: "ERROR: Invalid payload type" };
}
