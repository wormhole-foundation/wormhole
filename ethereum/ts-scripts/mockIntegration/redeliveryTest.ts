import * as wh from "@certusone/wormhole-sdk";
import {
  Implementation__factory,
  Migrations,
} from "@certusone/wormhole-sdk/lib/cjs/ethers-contracts";
import { getSignatureSetData } from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import { LogMessagePublishedEvent } from "../../sdk/src";
import {
  ChainInfo,
  getCoreRelayer,
  getCoreRelayerAddress,
  getMockIntegration,
  getMockIntegrationAddress,
  getOperatingChains,
  getRelayProvider,
  getRelayProviderAddress,
  init,
  loadChains,
} from "../helpers/env";
import * as grpcWebNodeHttpTransport from "@improbable-eng/grpc-web-node-http-transport";
import { BigNumber } from "ethers";
import { wait } from "../helpers/utils";

init();
const chains = getOperatingChains();

async function run(
  sourceChain: ChainInfo,
  targetChain: ChainInfo,
  nonce: number,
  sourceTxHash: string
) {
  const coreRelayer = getCoreRelayer(sourceChain);
  const relayProvider = await coreRelayer.getDefaultRelayProvider();

  const relayQuote = await (
    await coreRelayer.quoteGas(targetChain.chainId, 2000000, relayProvider)
  ).add(10000000000);

  const rx = await coreRelayer
    .resend(
      {
        sourceChain: sourceChain.chainId,
        sourceNonce: nonce,
        sourceTxHash: sourceTxHash,
        targetChain: targetChain.chainId,
        deliveryIndex: 1,
        multisendIndex: 0,
        newMaxTransactionFee: relayQuote,
        newReceiverValue: BigNumber.from(0),
        newRelayParameters: new Uint8Array(),
      },
      nonce,
      relayProvider,
      { value: relayQuote, gasLimit: 1000000 }
    )
    .then(wait);
  console.log(rx);
}

async function main() {
  await run(getChainById(6), getChainById(14), 1, process.argv[2]);
}

console.log("Start!");
main().then(() => console.log("Done!"));

/* Helpers */

export function getChainById(id: number | string): ChainInfo {
  id = Number(id);
  const chain = chains.find((c) => c.chainId === id);
  if (!chain) {
    throw new Error("chainId not found, " + id);
  }
  return chain;
}
