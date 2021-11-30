import { ChainId } from "@certusone/wormhole-sdk";
import swapPools from "../swapPools.json";

export default function getSwapPool(
  sourceChain: ChainId,
  targetChain: ChainId
) {
  const map = swapPools as any;
  const holder = map[sourceChain]?.[targetChain];
  console.log("read swap pool", holder);
  return holder;
}
