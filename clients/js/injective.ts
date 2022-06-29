import { getNetworkInfo, Network } from "@injectivelabs/networks";
import { ChainGrpcClient } from "@injectivelabs/sdk-ts/dist/client/chain/ChainGrpcClient";
export async function query() {
  const network = getNetworkInfo(Network.TestnetK8s);
  const injectiveAddress = "inj180rl9ezc4389t72pc3vvlkxxs5d9jx60w9eeu3";
  const chainClient = new ChainGrpcClient(network.sentryGrpcApi);

  const balances = await chainClient.bank.fetchBalances(injectiveAddress);
  const injBalance = balances.balances.find(
    (balance) => balance.denom === "inj"
  );

  console.log(injBalance);
}
