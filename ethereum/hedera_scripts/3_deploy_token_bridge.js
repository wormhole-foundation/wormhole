require("dotenv").config({ path: ".env" });
const {
  AccountId,
  PrivateKey,
  Client,
  ContractFunctionParameters,
} = require("@hashgraph/sdk");
const Web3 = require("web3");
// const web3 = new Web3("ws://localhost:8545");
const web3 = new Web3();

const deployer = require("./deploy.js");

// CONFIG
const chainId = process.env.BRIDGE_INIT_CHAIN_ID;
const governanceChainId = process.env.BRIDGE_INIT_GOV_CHAIN_ID;
const governanceContract = process.env.BRIDGE_INIT_GOV_CONTRACT; // bytes32
const WETH = process.env.BRIDGE_INIT_WETH;
const WORMHOLE_ADDRESS = process.env.WORMHOLE_ADDRESS;
const finality = process.env.BRIDGE_INIT_FINALITY;

// Configure accounts and client
const operatorId = AccountId.fromString(process.env.OPERATOR_ID);
const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY);

const client = Client.forTestnet().setOperator(operatorId, operatorKey);

async function main() {
  const TokenImplementation = require("../build/contracts/TokenImplementation.json");
  const TokenImplementationAddress = await deployer.deploy(
    client,
    "TokenImplementation",
    TokenImplementation.bytecode,
    100000,
    new ContractFunctionParameters()
  );

  const BridgeSetup = require("../build/contracts/BridgeSetup.json");
  const BridgeSetupAddress = await deployer.deploy(
    client,
    "BridgeSetup",
    BridgeSetup.bytecode,
    100000,
    new ContractFunctionParameters()
  );

  const BridgeImplementation = require("../build/contracts/BridgeImplementation.json");
  const BridgeImplementationAddress = await deployer.deploy(
    client,
    "BridgeImplementation",
    BridgeImplementation.bytecode,
    100000,
    new ContractFunctionParameters()
  );

  console.log("generating setup initialization data...");

  const setup = new web3.eth.Contract(BridgeSetup.abi, BridgeSetupAddress);
  const initData = setup.methods
    .setup(
      BridgeImplementationAddress,
      chainId,
      WORMHOLE_ADDRESS,
      governanceChainId,
      governanceContract,
      TokenImplementationAddress,
      WETH,
      finality
    )
    .encodeABI();

  const TokenBridge = require("../build/contracts/TokenBridge.json");
  const params = new ContractFunctionParameters()
    .addAddress(BridgeSetupAddress)
    .addBytes(new Uint8Array(Buffer.from(initData.substring(2), "hex")));

  await deployer.deploy(
    client,
    "TokenBridge",
    TokenBridge.bytecode,
    200000,
    params
  );
  console.log("TokenBridge deploy complete");
}

main();
