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
const chainId = process.env.BRIDGE_INIT_CHAIN_ID;
const governanceChainId = process.env.BRIDGE_INIT_GOV_CHAIN_ID;
const governanceContract = process.env.BRIDGE_INIT_GOV_CONTRACT; // bytes32
const WETH = process.env.BRIDGE_INIT_WETH;
const WORMHOLE_ADDRESS = process.env.WORMHOLE_ADDRESS;

// Configure accounts and client
const operatorId = AccountId.fromString(process.env.OPERATOR_ID);
const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY);

const client = Client.forTestnet().setOperator(operatorId, operatorKey);

async function main() {
  const NFTImplementation = require("../build/contracts/NFTImplementation.json");
  const NFTImplementationAddress = await deployer.deploy(client, "NFTImplementation", NFTImplementation.bytecode, 100000, new ContractFunctionParameters());

  const NFTBridgeSetup = require("../build/contracts/NFTBridgeSetup.json");
  const NFTBridgeSetupAddress = await deployer.deploy(client, "NFTBridgeSetup", NFTBridgeSetup.bytecode, 100000, new ContractFunctionParameters());
  
  const NFTBridgeImplementation = require("../build/contracts/NFTBridgeImplementation.json");
  const NFTBridgeImplementationAddress = await deployer.deploy(client, "NFTBridgeImplementation", NFTBridgeImplementation.bytecode, 100000, new ContractFunctionParameters());

  console.log("generating setup initialization data...");

  const setup = new web3.eth.Contract(NFTBridgeSetup.abi, NFTBridgeSetupAddress);
  const initData = setup.methods.setup(
    NFTBridgeImplementationAddress,
      chainId,
      WORMHOLE_ADDRESS,
      governanceChainId,
      governanceContract,
      NFTImplementationAddress
  ).encodeABI();

  const NFTBridgeEntrypoint = require("../build/contracts/NFTBridgeEntrypoint.json");
  const params = new ContractFunctionParameters()
    .addAddress(NFTBridgeSetupAddress)
    .addBytes(new Uint8Array(Buffer.from(initData.substring(2), "hex")));

  await deployer.deploy(client, "NFTBridgeEntrypoint", NFTBridgeEntrypoint.bytecode, 200000, params);
  console.log("NFT Bridge deploy complete")
}

main();
