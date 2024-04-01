import {
  init,
  loadChains,
  ChainInfo,
  writeOutputFiles,
  getMockIntegration,
} from "../helpers/env";

const processName = "readMockIntegrationState";
init();
const chains = loadChains();

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

type MockIntegrationContractState = {
  chainId: number;
  contractAddress: string;
  messageHistory: string[][];
  registeredContracts: { chainId: number; contract: string }[];
};

async function readState(
  chain: ChainInfo
): Promise<MockIntegrationContractState | null> {
  console.log(
    "Gathering mock integration contract status for chain " + chain.chainId
  );

  try {
    const mockIntegration = getMockIntegration(chain);
    const contractAddress = mockIntegration.address;
    const messageHistory = await mockIntegration.getMessageHistory();
    const registeredContracts: { chainId: number; contract: string }[] = [];

    for (const chainInfo of chains) {
      registeredContracts.push({
        chainId: chainInfo.chainId,
        contract: await mockIntegration.getRegisteredContract(
          chainInfo.chainId
        ),
      });
    }

    return {
      chainId: chain.chainId,
      contractAddress,
      messageHistory,
      registeredContracts,
    };
  } catch (e) {
    console.error(e);
    console.log("Failed to gather status for chain " + chain.chainId);
  }

  return null;
}

function printState(state: MockIntegrationContractState) {
  console.log("");
  console.log("MockRelayerIntegration: ");
  printFixed("Chain ID: ", state.chainId.toString());
  printFixed("Contract Address:", state.contractAddress);

  console.log("");

  printFixed("Registered Contracts", "");
  state.registeredContracts.forEach((x) => {
    printFixed("  Chain: " + x.chainId, JSON.stringify(x.contract));
  });
  console.log("");

  console.log("MessageHistory");
  console.log(state.messageHistory);
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
