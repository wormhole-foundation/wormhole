import type { SuiClientTypes } from "@mysten/sui/client";
import { NETWORKS } from "../../consts/networks";
import { getObjectFields } from "../../sdk/sui";
import { getProvider } from "./utils";
import {
  ChainId,
  Network,
  chainIdToChain,
  contracts,
} from "@wormhole-foundation/sdk";

export async function queryRegistrationsSui(
  network: Network,
  module: "Core" | "NFTBridge" | "TokenBridge"
): Promise<Object> {
  const n = NETWORKS[network]["Sui"];
  const provider = getProvider(network, n.rpc);
  let state_object_id: string;

  switch (module) {
    case "TokenBridge":
      state_object_id = contracts.tokenBridge(network, "Sui");
      if (state_object_id === undefined) {
        throw Error(`Unknown token bridge contract on ${network} for Sui`);
      }
      break;
    default:
      throw new Error(`Invalid module: ${module}`);
  }

  const state = await getObjectFields(provider, state_object_id);
  const emitterRegistryId = state!.emitter_registry.id;

  const results: { [key: string]: string } = {};
  let cursor: string | null = null;
  let hasNextPage = true;
  while (hasNextPage) {
    const page: SuiClientTypes.ListDynamicFieldsResponse =
      await provider.listDynamicFields({
        parentId: emitterRegistryId,
        cursor,
      });
    for (const field of page.dynamicFields) {
      const entry = await provider.getObject({
        objectId: field.fieldId,
        include: { json: true },
      });
      const json = entry.object.json as any;
      const chainId = json?.name as ChainId;
      // The gRPC JSON representation encodes the `vector<u8>` emitter address as
      // base64 under `value.value.data`.
      const dataB64: string | undefined = json?.value?.value?.data;
      if (chainId === undefined || dataB64 === undefined) {
        continue;
      }
      results[chainIdToChain(chainId)] = Buffer.from(
        dataB64,
        "base64"
      ).toString("hex");
    }
    hasNextPage = page.hasNextPage;
    cursor = page.cursor;
  }

  return results;
}
