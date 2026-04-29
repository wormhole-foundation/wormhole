import { getCosmWasmClient } from "@sei-js/core";
import { NETWORKS } from "../../consts/networks";
import { Chain, Network, chains, contracts } from "@wormhole-foundation/sdk-base";

export async function queryRegistrationsSei(
  network: Network,
  module: "Core" | "NFTBridge" | "TokenBridge"
): Promise<Object> {
  const chain: Chain = "Sei";
  const n = NETWORKS[network][chain];

  let target_contract: string | undefined;

  switch (module) {
    case "TokenBridge":
      target_contract = contracts.tokenBridge.get(network, chain);
      break;
    case "NFTBridge":
      target_contract = contracts.nftBridge.get(network, chain);
      break;
    default:
      throw new Error(`Invalid module: ${module}`);
  }

  if (!target_contract) {
    throw new Error(`Contract for ${module} on ${network} does not exist`);
  }

  if (n.rpc === undefined) {
    throw new Error(`RPC for ${module} on ${network} does not exist`);
  }

  // Create a CosmWasmClient
  const client = await getCosmWasmClient(n.rpc);

  // Query the bridge registration for all the chains in parallel.
  const registrations = await Promise.all(
    chains
      .filter((c_name) => c_name !== chain)
      .map(async (c_name) => [
        c_name,
        await (async () => {
          let query_msg = {
            chain_registration: {
              chain: c_name,
            },
          };

          let result = null;
          try {
            result = await client.queryContractSmart(
              target_contract as string,
              query_msg
            );
          } catch {
            // Not logging anything because a chain not registered returns an error.
          }

          return result;
        })(),
      ])
  );

  const results: { [key: string]: string } = {};
  for (let [c_name, queryResponse] of registrations) {
    if (queryResponse) {
      results[c_name] = Buffer.from(queryResponse.address, "base64").toString(
        "hex"
      );
    }
  }
  return results;
}
