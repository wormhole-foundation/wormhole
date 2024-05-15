import { transferFromSui } from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import {
  executeTransactionBlock,
  getProvider,
  getSigner,
  setMaxGasBudgetDevnet,
} from "./utils";
import { tryNativeToUint8Array } from "@certusone/wormhole-sdk/lib/esm/utils";
import {
  Chain,
  Network,
  chainToChainId,
  contracts,
} from "@wormhole-foundation/sdk-base";

export async function transferSui(
  dstChain: Chain,
  dstAddress: string,
  tokenAddress: string,
  amount: string,
  network: Network,
  rpc: string
) {
  const core = contracts.coreBridge(network, "Sui");
  if (!core) {
    throw Error("Core bridge object ID is undefined");
  }
  const token_bridge = contracts.tokenBridge.get(network, "Sui");
  if (!token_bridge) {
    throw new Error("Token bridge object ID is undefined");
  }
  const provider = getProvider(network, rpc);
  const signer = getSigner(provider, network);
  const owner = await signer.getAddress();
  const coinType = tokenAddress === "native" ? "0x2::sui::SUI" : tokenAddress;
  const coins = (
    await provider.getCoins({
      owner,
      coinType,
    })
  ).data;
  const tx = await transferFromSui(
    provider,
    core,
    token_bridge,
    coins,
    coinType,
    BigInt(amount),
    chainToChainId(dstChain),
    tryNativeToUint8Array(dstAddress, chainToChainId(dstChain))
  );
  setMaxGasBudgetDevnet(network, tx);
  const result = await executeTransactionBlock(signer, tx);
  console.log(JSON.stringify(result));
}
