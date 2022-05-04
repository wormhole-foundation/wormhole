const {
  ContractCreateFlow,
} = require("@hashgraph/sdk");

// Based on example for ContractCreateFlow from https://docs.hedera.com/guides/docs/sdks/smart-contracts/create-a-smart-contract#methods
async function deploy(client, contractName, contractBytecode, gas, constructorFunctionParameters) {
  console.log("deploying " + contractName + ", gas: " + gas + ", byteCodeLen: " + contractBytecode.length)

  //Create the transaction
  const contractCreate = new ContractCreateFlow()
      .setGas(gas)
      .setBytecode(contractBytecode)
      .setConstructorParameters(constructorFunctionParameters)

  //Sign the transaction with the client operator key and submit to a Hedera network
  const txResponse = contractCreate.execute(client)

  //Get the receipt of the transaction
  const receipt = (await txResponse).getReceipt(client)

  //Get the new contract ID
  const contractId = (await receipt).contractId
  const contractAddress = "0x" + contractId.toSolidityAddress()

  console.log("deployed " + contractName + ", contractId: " + contractId + ", contractAddress: " + contractAddress)
  return contractAddress
}

module.exports = {deploy}