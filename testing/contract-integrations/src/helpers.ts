import {
  attestFromEth,
  ChainId,
  parseSequenceFromLogEth,
} from "@certusone/wormhole-sdk";
import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import { getBridgeAddressForChain } from "./consts";
import { getSignerForChain, getTokenBridgeAddressForChain } from "./consts";

setDefaultWasm("node");

export async function attestEvm(
  originChain: ChainId,
  originAsset: string
): Promise<string> {
  const signer = getSignerForChain(originChain);
  const receipt = await attestFromEth(
    getTokenBridgeAddressForChain(originChain),
    signer,
    originAsset
  );
  const sequence = parseSequenceFromLogEth(
    receipt,
    getBridgeAddressForChain(originChain)
  );
  return sequence;
}
