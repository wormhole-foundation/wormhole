import { ChainId } from "@wormhole-foundation/sdk";
import fs from "fs";
import { join } from "path";
import { execSync } from "child_process";

export interface SerializableDeployment {
  chainId: ChainId;
  address: string;
};


export interface EvmSerializableDeployment extends SerializableDeployment {
  deployTxid: string;
  constructorArgs?: any[];
};

export type Network = "Mainnet" | "Testnet";

enum ContractWrites {
  NoChanges,
}

type ContractsJson = Record<string, SerializableDeployment[]>;
class ContractAddresses {
  static contractAddresses: ContractAddresses;
  public static get(env: Network): ContractAddresses {
    if (this.contractAddresses === undefined) this.contractAddresses = new ContractAddresses(env);

    return this.contractAddresses;
  }

  public readonly path: string;
  private readonly contracts: ContractsJson;

  private constructor(env: string) {
    this.path = `./config/${env}/contracts.json`;
    const contractsFile = fs.readFileSync(this.path, "utf8");
    this.contracts = JSON.parse(contractsFile);
  }

  updateContracts(newContracts: ContractsJson) {
    if (Object.values(newContracts).every((deployments) => deployments.length === 0)) {
      return ContractWrites.NoChanges;
    }

    const commit = getCurrentCommit();
    for (const [key, newDeployments] of Object.entries(newContracts)) {
      const newDeploymentsWithCommit = newDeployments.map((deployment) => ({...deployment, commit}))
      const savedDeployments = this.contracts[key] ?? [];
      this.contracts[key] = this.mergeContractAddresses(
        savedDeployments,
        newDeploymentsWithCommit,
      );
    }

    const serializedContracts = this.serialize();
    fs.writeFileSync(this.path, serializedContracts);
    return serializedContracts;
  }

  private updateContractAddress(
    arr: SerializableDeployment[],
    newAddress: SerializableDeployment,
  ) {
    const idx = arr.findIndex((a) => a.chainId === newAddress.chainId);
    if (idx === -1) {
      arr.push(newAddress);
    } else {
      arr[idx] = newAddress;
    }
  }

  private mergeContractAddresses(
    arr: SerializableDeployment[],
    newAddresses: SerializableDeployment[],
  ): SerializableDeployment[] {
    const newArray = [...arr];
    for (const newAddress of newAddresses) {
      this.updateContractAddress(newArray, newAddress);
    }
    return newArray;
  }

  loadAddress(name: string, chain: ChainId) {
    return this.contracts[name]?.find((a) => a.chainId === chain)?.address;
  }

  loadContract(name: string): SerializableDeployment[] {
    const addresses = this.contracts[name];
    if (addresses === undefined) throw new Error(`Failed to find ${name} in contracts file`);
    return addresses;
  }

  tryLoadContract(name: string): SerializableDeployment[] | undefined {
    return this.contracts[name];
  }

  serialize() {
    return JSON.stringify(this.contracts, undefined, 2);
  }
}

export function loadAddress(contractName: string, chain: ChainId, env: Network) {
  const contractAddresses = ContractAddresses.get(env);
  return contractAddresses.loadAddress(contractName, chain);
}

export function loadContractsFromFile(contractName: string, env: Network) {
  const contractAddresses = ContractAddresses.get(env);
  return contractAddresses.loadContract(contractName);
}

export function tryLoadContractsFromFile(contractName: string, env: Network) {
  const contractAddresses = ContractAddresses.get(env);
  return contractAddresses.tryLoadContract(contractName);
}


function writeOutputFiles(output: unknown, processName: string, env: Network) {
  const commit = getCurrentCommit();
  const basePath = `./output/${env}/${processName}`;
  fs.mkdirSync(basePath, {
    recursive: true,
  });
  fs.writeFileSync(
    join(basePath, "lastrun.json"),
    JSON.stringify({commit, output}),
    { flag: "w" }
  );
  fs.writeFileSync(
    join(basePath, `${Date.now()}.json`),
    JSON.stringify({commit, output}),
    { flag: "w" }
  );
}


/**
 * Saves deployments using the (contract name, chain id) tuple as a key.
 * Fully overwrites old deployments for that particular key.
 */
export function saveDeployments(
  newContracts: ContractsJson,
  processName: string,
  env: Network,
) {
  writeOutputFiles(newContracts, processName, env);
  syncContractsJson(newContracts, env);
}

function syncContractsJson(newContracts: ContractsJson, env: Network) {
  const contractAddresses = ContractAddresses.get(env);

  const writtenFile = contractAddresses.updateContracts(newContracts);
  if (writtenFile === ContractWrites.NoChanges) {
    console.log("No changes to deployments.");
    return;
  }
}

let currentCommit: string;
function getCurrentCommit() {
  if (currentCommit !== undefined) return currentCommit;
  currentCommit = execSync("git rev-parse HEAD").toString().trim();
  return currentCommit;
}
