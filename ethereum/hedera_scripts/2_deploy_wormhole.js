require("dotenv").config({ path: ".env" });
const {
  AccountId,
  PrivateKey,
  Client,
  ContractFunctionParameters,
} = require("@hashgraph/sdk");
const Web3 = require("web3");
const web3 = new Web3("ws://localhost:8545");

const deployer = require("./deploy.js")

// CONFIG
const initialSigners = JSON.parse(process.env.INIT_SIGNERS);
const chainId = process.env.INIT_CHAIN_ID;
const governanceChainId = process.env.INIT_GOV_CHAIN_ID;
const governanceContract = process.env.INIT_GOV_CONTRACT; // bytes32

// Configure accounts and client
const operatorId = AccountId.fromString(process.env.OPERATOR_ID);
const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY);

const client = Client.forTestnet().setOperator(operatorId, operatorKey);

async function main() {
  const Setup = require("../build/contracts/Setup.json");
  const SetupAddress = await deployer.deploy(client, "Setup", Setup.bytecode, 100000, new ContractFunctionParameters());

  const Implementation = require("../build/contracts/Implementation.json");
  const ImplementationAddress = await deployer.deploy(client, "Implementation", Implementation.bytecode, 100000, new ContractFunctionParameters());

  console.log("generating setup initialization data...");
  const setup = new web3.eth.Contract(Setup.abi, SetupAddress);
  const initData = setup.methods
    .setup(
      ImplementationAddress,
      initialSigners,
      chainId,
      governanceChainId,
      governanceContract
    )
    .encodeABI();

  const Wormhole = require("../build/contracts/Wormhole.json");
  const params = new ContractFunctionParameters()
    .addAddress(SetupAddress)
    .addBytes(new Uint8Array(Buffer.from(initData.substring(2), "hex")));

  await deployer.deploy(client, "Wormhole", Wormhole.bytecode, 200000, params);
  console.log("Wormhole deploy complete")
}

main();
