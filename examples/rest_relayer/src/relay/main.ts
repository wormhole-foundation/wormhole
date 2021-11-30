import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { RelayerEnvironment, validateEnvironment } from "../configureEnv";
import { relayEVM } from "./evm";
import { relaySolana } from "./solana";
import { relayTerra } from "./terra";

const env: RelayerEnvironment = validateEnvironment();

function getChainConfigInfo(chainId: ChainId) {
  return env.supportedChains.find((x) => x.chainId === chainId);
}

function validateRequest(request: any, response: any) {
  const chainId = request.body?.chainId;
  const chainConfigInfo = getChainConfigInfo(chainId);
  const unwrapNative = request.body?.unwrapNative || false;

  if (!chainConfigInfo) {
    response.status(400).json({ error: "Unsupported chainId" });
    return;
  }
  const signedVAA = request.body?.signedVAA;
  if (!signedVAA) {
    response.status(400).json({ error: "signedVAA is required" });
  }

  //TODO parse & validate VAA.
  //TODO accept redeem native parameter

  return { chainConfigInfo, chainId, signedVAA, unwrapNative };
}

export async function relay(request: any, response: any) {
  console.log("Incoming request for relay: ", request.body);
  const { chainConfigInfo, chainId, signedVAA, unwrapNative } = validateRequest(
    request,
    response
  );

  try {
    if (isEVMChain(chainId)) {
      await relayEVM(
        chainConfigInfo,
        signedVAA,
        unwrapNative,
        request,
        response
      );
    } else if (chainId === CHAIN_ID_SOLANA) {
      await relaySolana(
        chainConfigInfo,
        signedVAA,
        unwrapNative,
        request,
        response
      );
    } else if (chainId === CHAIN_ID_TERRA) {
      await relayTerra(chainConfigInfo, signedVAA, request, response);
    } else {
      response.status(400).json({ error: "Improper chain ID" });
    }
  } catch (e) {
    console.log("Error while relaying");
    console.error(e);
    response.status(500).json({ error: "Unable to relay this request." });
  }
}
