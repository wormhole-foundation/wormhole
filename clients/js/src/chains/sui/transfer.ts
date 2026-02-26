import { transferFromSui } from "@certusone/wormhole-sdk/lib/esm/token_bridge/transfer";
import {
  executeTransactionBlock,
  getProvider,
  getSigner,
  setMaxGasBudgetDevnet,
} from "./utils";
import {
  Chain,
  Network,
  chainToChainId,
  contracts,
} from "@wormhole-foundation/sdk-base";
import { tryNativeToUint8Array } from "../../sdk/array";

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
  const client = getProvider(network, rpc);
  const signer = getSigner(client, network);
  const owner = signer.keypair.getPublicKey().toSuiAddress();
  const coinType = tokenAddress === "native" ? "0x2::sui::SUI" : tokenAddress;
  const coins = (
    await client.getCoins({
      owner,
      coinType,
    })
  ).data;
  const tx = await transferFromSui(
    client as any,
    core,
    token_bridge,
    coins,
    coinType,
    BigInt(amount),
    chainToChainId(dstChain),
    tryNativeToUint8Array(dstAddress, chainToChainId(dstChain))
  );
  setMaxGasBudgetDevnet(network, tx as any);
  const result = await executeTransactionBlock(signer, tx as any);
  console.log(JSON.stringify(result));
}
