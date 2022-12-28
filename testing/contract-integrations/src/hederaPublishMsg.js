const {
  AccountId,
  PrivateKey,
  Client,
  ContractExecuteTransaction,
  ContractFunctionParameters,
  ContractId,
} = require("@hashgraph/sdk");

(async () => {
  // TESTNET_RPC = process.env.TESTNET_RPC;
  // if (TESTNET_RPC === "") {
  //   console.error('Please set "TESTNET_RPC"');
  //   return;
  // }

  CONTRACT_ID = process.env.CONTRACT_ID;
  if (CONTRACT_ID === "") {
    console.error('Please set "CONTRACT_ID"');
    return;
  }

  OPERATOR_ID = process.env.OPERATOR_ID;
  if (OPERATOR_ID === "") {
    console.error('Please set "OPERATOR_ID"');
    return;
  }

  OPERATOR_PVKEY = process.env.OPERATOR_PVKEY;
  if (OPERATOR_PVKEY === "") {
    console.error('Please set "OPERATOR_PVKEY"');
    return;
  }

  CONSISTENCY_LEVEL = process.env.CONSISTENCY_LEVEL;
  if (CONSISTENCY_LEVEL === "") {
    console.error('Please set "CONSISTENCY_LEVEL"');
    return;
  }
  consistencyLevel = parseInt(CONSISTENCY_LEVEL, 10);

  console.log(
    "publishing a message to contract " +
      CONTRACT_ID +
      ", using consistency level " +
      consistencyLevel
  );

  // Hedera specific calls here

  // Configure accounts and client
  const operatorId = AccountId.fromString(process.env.OPERATOR_ID);
  const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY);
  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  // Create the transaction
  const transaction = new ContractExecuteTransaction()
    .setContractId(ContractId.fromString(CONTRACT_ID))
    .setGas(100_000)
    .setFunction(
      "publishMsg",
      new ContractFunctionParameters().addUint8(consistencyLevel)
    );

  //Sign with the client operator private key to pay for the transaction and submit the query to a Hedera network
  const txResponse = await transaction.execute(client);
  console.log("txResponse", txResponse);

  //Request the receipt of the transaction
  const receipt = await txResponse.getReceipt(client);
  console.log("receipt", receipt);

  //Get the transaction consensus status
  const transactionStatus = receipt.status;

  console.log("The transaction status is " + transactionStatus);

  // This is the old EVM code
  // const provider = new ethers.providers.JsonRpcProvider(TESTNET_RPC);
  // const signer = new ethers.Wallet(process.env.MNEMONIC, provider);
  // const contract = new ethers.Contract(
  //   PUBLISH_MSG_ADDRESS,
  //   [
  //     "function publishMsg(uint8 consistencyLevel) public payable returns (uint64 sequence)",
  //   ],
  //   signer
  // );
  // const tx = await contract.publishMsg(consistencyLevel);
  // const receipt = await tx.wait();
  // console.log(receipt);
})();
