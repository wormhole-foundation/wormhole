require("dotenv").config({ path: ".env" });
const {
  AccountId,
  PrivateKey,
  Client,
  ContractCallQuery,
  Hbar,
  AccountBalanceQuery,
} = require("@hashgraph/sdk");

// Configure accounts and client
const operatorId = AccountId.fromString(process.env.OPERATOR_ID);
const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY);
const client = Client.forTestnet().setOperator(operatorId, operatorKey);
const contractId = process.env.CONTRACT_ID;

(async () => {
  //Create the Query
  // has to be the contractId not the token Id
  // const contractId = "0.0.49346530"; //<-- from deployed token contract
  const query = new AccountBalanceQuery().setAccountId(operatorId);

  //Submit the query to a Hedera network
  const accountBalance = await query.execute(client);

  //Print the balance of hbars
  console.log(
    "The hbar account balance for this account is " + accountBalance.hbars
  );
  const functionCall = "totalSupply";
  // query the contract
  const contractQuery = await new ContractCallQuery()
    //Set the gas for the query
    .setGas(100000)
    //Set the contract ID to return the request for
    .setContractId(contractId)
    //Set the contract function to call
    .setFunction(functionCall)
    //Set the query payment for the node returning the request
    //This value must cover the cost of the request otherwise will fail
    .setQueryPayment(new Hbar(2));

  const functionQuery = await contractQuery.execute(client);

  // Need to decode depending on what is expected type for functionQueryResult
  // Get a value from the result at index 0
  // const message = getContractAddress.getBytes32(0);
  const functionQueryResult = functionQuery.getUint256(0);

  //Log the message
  // console.log("The contract message: " + message.toString("hex"));
  console.log(`The contract ${functionCall}: ${functionQueryResult}`);

  //Contract call query
  // const query = new ContractCallQuery()
  //   .setContractId(contractId)
  //   .setGas(300000)
  //   .setFunction("owner");

  // //Sign with the client operator private key to pay for the query and submit the query to a Hedera network
  // const contractCallResult = await query.execute(client);

  // // Get the function value depending on what your expected return type is
  // // const message = contractCallResult.getBytes32(0);
  // // const message = contractCallResult.getUint256(0);
  // // const message = contractCallResult.getUint32(0);
  // // const message = contractCallResult.getString(0);
  // const message = contractCallResult.getAddress(0);
  // console.log("contract message: " + message);

  // const nativeContract = new ContractCallQuery()
  //   .setContractId(contractId)
  //   .setGas(400000)
  //   // .setPayableAmount(200) // Increase if revert
  //   .setFunction("stupidsymbol", new ContractFunctionParameters());

  // const nativeContractTx = await nativeContract.execute(client);
  // // console.log("post execute");
  // // const nativeContractRx = await nativeContractTx.getRecord(client);
  // // console.log("post nativeContractRx", nativeContractRx);
  // const tokenIdSolidityAddr = nativeContractTx.getBytes32(0);
})();
