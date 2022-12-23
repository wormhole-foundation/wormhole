require("dotenv").config({ path: ".env" });
const {
  AccountId,
  PrivateKey,
  Client,
  ContractFunctionParameters,
} = require("@hashgraph/sdk");
const Web3 = require("web3");
const web3 = new Web3("ws://localhost:8545");

const deployer = require("./deploy.js");

// CONFIG
const WormholeAddress = process.env.WORMHOLE_ADDRESS;

// Configure accounts and client
const operatorId = AccountId.fromString(process.env.OPERATOR_ID);
const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY);

const client = Client.forTestnet().setOperator(operatorId, operatorKey);

async function main() {
  console.log("generating setup initialization data...");

  const PublishMsg = require("../../testing/contract-integrations/build/contracts/PublishMsg.json");
  const params = new ContractFunctionParameters().addAddress(WormholeAddress);

  await deployer.deploy(
    client,
    "PublishMsg",
    PublishMsg.bytecode,
    200000,
    params
  );
  console.log("PublishMsg deploy complete");
}

await main();
