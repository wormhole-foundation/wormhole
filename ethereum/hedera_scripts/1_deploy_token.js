require("dotenv").config({ path: ".env" });
const {
  AccountId,
  PrivateKey,
  Client,
  ContractFunctionParameters,
  ContractId,
  ContractInfoQuery,
  ContractExecuteTransaction,
} = require("@hashgraph/sdk");
const { ContractCreateFlow } = require("@hashgraph/sdk");

const Web3 = require("web3");
const web3 = new Web3("ws://localhost:2222");

// Configure accounts and client
const operatorId = AccountId.fromString(process.env.OPERATOR_ID);
const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY);

const client = Client.forTestnet().setOperator(operatorId, operatorKey);

function encodeFunctionCall(functionName, parameters, abi) {
  const functionAbi = abi.find(
    (func) => func.name === functionName && func.type === "function"
  );
  console.log("encodeFunctionCall", functionName, parameters);
  const encodedParameters = web3.eth.abi.encodeFunctionCall(
    functionAbi,
    parameters
  );
  console.log("encodedParameters:", encodedParameters);
  const encodedParametersHex = encodedParameters.slice(2);
  return Buffer.from(encodedParametersHex, "hex");
}

async function queryContractInfo(contractName, contractAddress) {
  const contractId = ContractId.fromEvmAddress(
    0,
    0,
    contractAddress.substring(2)
  );
  console.log(
    "Querying " +
      contractName +
      " contract info, contract address: " +
      contractAddress +
      ", contractId " +
      contractId
  );
}
// This fails due to gas.
async function queryToken(contractName, tokenAddress) {
  const contractId = ContractId.fromEvmAddress(0, 0, tokenAddress.substring(2));
  console.log(
    "Getting token info on " +
      contractName +
      ", token address: " +
      tokenAddress +
      ", contractId " +
      contractId
  );
  //   const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  const query = new ContractInfoQuery().setContractId(contractId);

  //Sign the query with the client operator private key and submit to a Hedera network
  const info = await query.execute(client);
  console.log("contract info for " + contractName + ": %o", info);
}

async function main() {
  const TokenImplementation = require("../build/contracts/HederaTokenImplementation.json");

  //Create the transaction
  const contractCreate = new ContractCreateFlow()
    .setGas(100000)
    .setBytecode(TokenImplementation.bytecode);

  //Sign the transaction with the client operator key and submit to a Hedera network
  const txResponse = await contractCreate.execute(client);

  //Get the receipt of the transaction
  const receipt = await txResponse.getReceipt(client);
  //Get the new contract ID
  const contractId = receipt.contractId;
  // This is the contract Id you use to query the new hedera token
  console.log("contractId=", contractId.toString());

  // fake input params for token
  const tokenName = "spamToken";
  const tokenSymbol = "SPAM";
  const tokenDecimals = 8;
  const tokenSequence = 1;
  const tokenOwner = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc4";
  const tokenChainId = 2;
  const tokenNativeContract =
    "0x05416460deb76d57af601be17e777b93592d8d4d4a4096c57876a91c84f4a712";

  // initialize the token
  const initialize = new ContractExecuteTransaction()
    .setContractId(contractId)
    .setGas(400000)
    .setPayableAmount(200) // Increase if revert
    .setFunction(
      "initialize",
      new ContractFunctionParameters()
        .addString(tokenName)
        .addString(tokenSymbol)
        .addUint8(tokenDecimals)
        .addUint64(tokenSequence)
        .addAddress(tokenOwner)
        .addUint16(tokenChainId)
        .addBytes32(Web3.utils.hexToBytes(tokenNativeContract))
    );

  const createTokenTx = await initialize.execute(client);
  console.log("post execute");
  const createTokenRx = await createTokenTx.getRecord(client);
  console.log("post createTokenTx", createTokenRx);
  console.log(Buffer.from(createTokenRx.transactionHash).toString("base64"));
  console.log(createTokenRx.transactionId);
  console.log(createTokenRx.contractFunctionResult.contractId);
}

main();
