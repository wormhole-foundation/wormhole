import { AptosAccount, TxnBuilderTypes, BCS, AptosClient, FaucetClient } from "aptos";
import { aptosAccountObject } from "./constants";
import fs from 'fs';
import sha3 from 'js-sha3';
import { Serializer } from "aptos/dist/transaction_builder/bcs";

export const NODE_URL = "http://0.0.0.0:8080/v1";
export const FAUCET_URL = "http://0.0.0.0:8081";

const client = new AptosClient(NODE_URL);
const faucetClient = new FaucetClient(NODE_URL, FAUCET_URL);

class Module {
  bytecode: Uint8Array;

  constructor(bytecode: Uint8Array) {
    this.bytecode = bytecode;
  };

  serialize(serializer: Serializer): void {
    serializer.serializeBytes(this.bytecode);
  }
}

/** Publish a new module to the blockchain within the specified account */
export async function publishModule(accountFrom: AptosAccount, deployer: 'native' | 'deployer_contract', packageMetadata: Uint8Array, modules: Module[]): Promise<string> {
  const serializer = new BCS.Serializer();
  serializer.serializeU32AsUleb128(modules.length);
  modules.forEach(module => module.serialize(serializer));
  const serializedModules = serializer.getBytes();

  const packageMetadataSerializer = new BCS.Serializer();
  packageMetadataSerializer.serializeBytes(packageMetadata)
  const serializedPackageMetadata = packageMetadataSerializer.getBytes();

  let contract_address: string;
  let contract_fn: string;

  if (deployer === 'native') {
    contract_address = '0x1::code';
    contract_fn = 'publish_package_txn';
  } else {
    contract_address = `${accountFrom.address()}::deployer`;
    contract_fn = 'deploy_derived';
  }

  const moduleBundlePayload = new TxnBuilderTypes.TransactionPayloadEntryFunction(
    TxnBuilderTypes.EntryFunction.natural(
      contract_address,
      contract_fn,
      [],
      [
        serializedPackageMetadata,
        serializedModules
      ]
    )
  );

  const [{ sequence_number: sequenceNumber }, chainId] = await Promise.all([
    client.getAccount(accountFrom.address()),
    client.getChainId(),
  ]);
  const rawTxn = new TxnBuilderTypes.RawTransaction(
    TxnBuilderTypes.AccountAddress.fromHex(accountFrom.address()),
    BigInt(sequenceNumber),
    moduleBundlePayload,
    BigInt(5000), //max gas to be used
    BigInt(1), //price per unit gas
    BigInt(Math.floor(Date.now() / 1000) + 10),
    new TxnBuilderTypes.ChainId(chainId),
  );

  const sim = await client.simulateTransaction(accountFrom, rawTxn);
  sim.forEach((tx) => {
    if (!tx.success) {
      console.error(JSON.stringify(tx, null, 2));
      throw new Error(`Transaction failed: ${tx.vm_status}`);
    }
  });
  const bcsTxn = AptosClient.generateBCSTransaction(accountFrom, rawTxn);
  const transactionRes = await client.submitSignedBCSTransaction(bcsTxn);

  return transactionRes.hash;
}

async function deploy(accountFrom: AptosAccount, deployer: 'native' | 'deployer_contract', buildDir: string, moduleNames: string[]) {
  const modulesDir = `${buildDir}/bytecode_modules`;

  const modules = moduleNames
    .map(file => new Module(fs.readFileSync(`${modulesDir}/${file}.mv`)))

  const packageMetaData = fs.readFileSync(`${buildDir}/package-metadata.bcs`);

  console.log(`Publishing ${modules.length} modules`);

  const hash = await publishModule(accountFrom, deployer, packageMetaData, modules);
  console.log(`Transaction hash: ${hash}`);
  let _tx = await client.waitForTransactionWithResult(hash);
}


async function main() {
  const accountFrom = AptosAccount.fromAptosAccountObject(aptosAccountObject)

  const coins = 100000;
  await faucetClient.fundAccount(accountFrom.address(), coins);
  console.log(`Funded account with ${coins} coins`);

  // Phase 1, deploy deployer.

  await deploy(accountFrom, 'native', '../deployer/build/Deployer', ['deployer']);

  // Phase 2, deploy core modules.

  const wormholeAccount = sha3.sha3_256(Buffer.concat([accountFrom.address().toBuffer(), Buffer.from('wormhole', 'ascii')]));
  console.log(`Deploying core contracts under wormhole account: ${wormholeAccount}`);


  // THIS HAS TO BE IN THE ORDER THAT `aptos move compile` OUTPUTS
  const coreModules = [
    "cursor",
    "u32",
    "u256",
    "u16",
    "deserialize",
    "guardian_pubkey",
    "structs",
    "state",
    "serialize",
    "vaa",
    "governance",
    "wormhole"
  ]

  await deploy(accountFrom, 'deployer_contract', '../contracts/build/Wormhole', coreModules);
}

if (require.main === module) {
  main().then((resp) => console.log(resp));
}
