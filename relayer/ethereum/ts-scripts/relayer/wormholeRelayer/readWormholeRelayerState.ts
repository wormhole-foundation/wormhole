import {
  init,
  ChainInfo,
  getWormholeRelayerAddress,
  getProvider,
  writeOutputFiles,
  getWormholeRelayer,
  getOperatingChains,
} from "../helpers/env";

const processName = "readWormholeRelayerContractState";
init();
const chains = getOperatingChains();

async function run() {
  console.log("Start! " + processName);

  const states: any = [];

  for (let i = 0; i < chains.length; i++) {
    const state = await readState(chains[i]);
    if (state) {
      printState(state);
      states.push(state);
    }
  }

  writeOutputFiles(states, processName);
}

type WormholeRelayerContractState = {
  chainId: number;
  contractAddress: string;
  defaultProvider: string;
  registeredContracts: { chainId: number; contract: string }[];
};

async function readState(
  chain: ChainInfo
): Promise<WormholeRelayerContractState | null> {
  console.log(
    "Gathering core relayer contract status for chain " + chain.chainId
  );

  try {
    const contractAddress = await getWormholeRelayerAddress(chain);
    console.log("Querying " + contractAddress);

    const coreRelayer = await getWormholeRelayer(chain, getProvider(chain));

    console.log("Querying default provider for code");
    const provider = getProvider(chain);
    const codeReceipt = await provider.getCode(contractAddress);
    console.log("Code: " + codeReceipt);

    const registeredContracts: { chainId: number; contract: string }[] = [];

    for (const chainInfo of chains) {
      registeredContracts.push({
        chainId: chainInfo.chainId,
        contract: (
          await coreRelayer.getRegisteredWormholeRelayerContract(
            chainInfo.chainId
          )
        ).toString(),
      });
    }

    const defaultProvider = await coreRelayer.getDefaultDeliveryProvider();
    return {
      chainId: chain.chainId,
      contractAddress,
      defaultProvider,
      registeredContracts,
    };
  } catch (e) {
    console.error(e);
    console.log("Failed to gather status for chain " + chain.chainId);
  }

  return null;
}

function printState(state: WormholeRelayerContractState) {
  console.log("");
  console.log("WormholeRelayer: ");
  printFixed("Chain ID: ", state.chainId.toString());
  printFixed("Contract Address:", state.contractAddress);
  printFixed("Default Provider:", state.defaultProvider);

  console.log("");

  printFixed("Registered WormholeRelayers", "");
  state.registeredContracts.forEach((x) => {
    printFixed("  Chain: " + x.chainId, x.contract);
  });
  console.log("");
}

function printFixed(title: string, content: string) {
  const length = 80;
  const spaces = length - title.length - content.length;
  let str = "";
  if (spaces > 0) {
    for (let i = 0; i < spaces; i++) {
      str = str + " ";
    }
  }
  console.log(title + str + content);
}

run().then(() => console.log("Done! " + processName));
