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
import { ChainConfigInfo, getRelayerEnvironment } from "../configureEnv";
import { RelayResult, Status } from "../helpers/redisHelper";
import { getLogger, getScopedLogger, ScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import { _undef } from "../xRaydium/scripts/lib";

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
  const parsedVAA: BaseVAA = parse_vaa(hexToUint8Array(signedVAA));
  console.log("parsedVAA.payload[0] is: ", parsedVAA.payload[0]);
  if (parsedVAA.payload[0] === 3) {
    const transferPayload = parseTransferPayload(
      Buffer.from(parsedVAA.payload)
    );
    const chainConfigInfo = getChainConfigInfo(transferPayload.targetChain);
    const solanaChainConfigInfo = getChainConfigInfo(CHAIN_ID_SOLANA);
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
    if (!solanaChainConfigInfo) {
      logger.error("Failed to find solana chainId");
      return {
        status: Status.FatalError,
        result: "Fatal Error: solana not supported",
      };
    }
    if (isEVMChain(transferPayload.targetChain)) {
      console.log("in relay.ts EVM Chain");
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
        solanaChainConfigInfo,
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
      console.log("in relay.ts CHAIN_ID_SOLANA");
      let rResult: RelayResult = { status: Status.Error, result: "" };
      const emitterChainConfig = getChainConfigInfo(parsedVAA.emitter_chain);

      if (!chainConfigInfo) {
        logger.error(
          "relay: improper emitter chain ID: " + parsedVAA.emitter_chain
        );
        return {
          status: Status.FatalError,
          result:
            "Fatal Error: emitter chain " +
            parsedVAA.emitter_chain +
            " not supported",
        };
      }
      const retVal = await relaySolana(
        chainConfigInfo,
        _undef(emitterChainConfig),
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

export interface BaseVAA {
  version: number;
  guardianSetIndex: number;
  timestamp: number;
  nonce: number;
  emitter_chain: ChainId;
  emitter_address: Uint8Array; // 32 bytes
  sequence: number;
  consistency_level: number;
  payload: Uint8Array;
}
