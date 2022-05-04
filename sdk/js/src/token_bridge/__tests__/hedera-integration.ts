import {
  AccountBalanceQuery,
  AccountCreateTransaction,
  AccountId,
  ContractCallQuery,
  ContractExecuteTransaction,
  ContractFunctionParameters,
  ContractId,
  ContractInfoQuery,
  Hbar,
  LocalProvider,
  PrivateKey,
  Status,
  TokenAssociateTransaction,
  TokenCreateTransaction,
  TokenId,
  TokenSupplyType,
  TokenType,
  Wallet,
} from "@hashgraph/sdk";
import { expect, jest, test } from "@jest/globals";
import { Bridge__factory } from "../../ethers-contracts";
import {
  CHAIN_ID_HEDERA,
  textToHexString,
  tryHexToNativeAssetString,
  tryNativeToHexString,
} from "../../utils";

const { Client } = require("@hashgraph/sdk");
require("dotenv").config({ path: ".env" });

const Web3 = require("web3");
const web3 = new Web3();

// const OPERATOR_ID: string = "0.0.34399286";
// const OPERATOR_KEY: string = "302e020100300506032b657004220420af71a8a658dbbd297c131dedf7bb24dce87ea527d7e9a862b43a918bd5e337af";
// const HEDERA_NETWORK = "testnet";
const OPERATOR_ID = process.env.OPERATOR_ID || "";
const OPERATOR_KEY = process.env.OPERATOR_KEY || "";
const HEDERA_NETWORK = process.env.HEDERA_NETWORK || "";
const INIT_SIGNERS = ["0x13947Bd48b18E53fdAeEe77F3473391aC727C638"];
const WH_NAME = "Wormhole";
const WH_ADDR = "0000000000000000000000000000000002dc28a4";
const TB_NAME = "TokenBridge";
const TB_ADDR = "0000000000000000000000000000000002dc2a0e";
const TOKEN_IMPL = "0x0000000000000000000000000000000002dc2a08";

jest.setTimeout(60000);

function getClient() {
  const operatorId = AccountId.fromString(OPERATOR_ID);
  const operatorKey = PrivateKey.fromString(OPERATOR_KEY);

  // If we weren't able to grab it, we should throw a new error
  if (operatorId == null || operatorKey == null) {
    throw new Error("Both Operator ID and Operator Key must be present");
  }

  // Create a connection to the Hedera network
  const client = Client.forTestnet();
  expect(client).toBeDefined();

  client.setOperator(operatorId, operatorKey);
  return client;
}

function getWallet(): Wallet {
  const wallet = new Wallet(OPERATOR_ID, OPERATOR_KEY, new LocalProvider());
  return wallet;
}

function encodeFunctionCall(functionName: any, parameters: any, abi: any) {
  const functionAbi = abi.find(
    (func: any) => func.name === functionName && func.type === "function"
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

function decodeFunctionResult(functionName: any, resultAsBytes: any, abi: any) {
  console.log("decodeFunctionResult:", functionName, resultAsBytes, abi);
  const functionAbi = abi.find((func: any) => func.name === functionName);
  const functionParameters = functionAbi.outputs;
  console.log("functionParameters", functionParameters);
  const resultHex = "0x".concat(Buffer.from(resultAsBytes).toString("hex"));
  console.log("resultHex", resultHex);
  const result = web3.eth.abi.decodeParameters(functionParameters, resultHex);
  console.log("returning", result);
  return result;
}

test("Hedera Test Account balance", async () => {
  const client = getClient();
  expect(client).toBeDefined();

  //Verify the account balance
  const accountBalance = await new AccountBalanceQuery()
    .setAccountId(OPERATOR_ID)
    .execute(client);

  console.log(
    "The new account balance is: " +
      accountBalance.hbars.toTinybars() +
      " tinybar."
  );
});

test("Hedera Test Core Bridge Chain ID", async () => {
  const client = getClient();
  expect(client).toBeDefined();

  const contractId = ContractId.fromEvmAddress(0, 0, WH_ADDR);
  console.log(
    "Querying " +
      WH_NAME +
      " contract info, contract address: " +
      WH_ADDR +
      ", contractId " +
      contractId
  );

  const query = new ContractInfoQuery().setContractId(contractId);

  //Sign the query with the client operator private key and submit to a Hedera network
  const info = await query.execute(client);
  // console.log("contract info for " + WH_NAME + ": %o", info);
  expect(info.contractAccountId).toBe(WH_ADDR);

  const jsonFile = "../../../../../ethereum/build/contracts/Getters.json";
  const json = require(jsonFile);
  const param = encodeFunctionCall("chainId", [], json.abi);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  const decodedChainId: number = parseInt(
    decodeFunctionResult("chainId", result.bytes, json.abi)[0]
  );
  console.log("chainId: ", decodedChainId);
  expect(decodedChainId).toBe(CHAIN_ID_HEDERA);
});

test("Hedera Test Query Guardian Set Index", async () => {
  const client = getClient();
  expect(client).toBeDefined();

  const jsonFile = "../../../../../ethereum/build/contracts/Getters.json";
  const json = require(jsonFile);

  const contractId = ContractId.fromEvmAddress(0, 0, WH_ADDR);
  // console.log(
  //   "Querying guardian set index, contract address: " + WH_ADDR,
  //   ", contractId " + contractId
  // );

  const param = encodeFunctionCall("getCurrentGuardianSetIndex", [], json.abi);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  const gsidx = parseInt(
    decodeFunctionResult(
      "getCurrentGuardianSetIndex",
      result.bytes,
      json.abi
    )[0]
  );
  console.log("current guardian set index:", gsidx);
  expect(gsidx).toBe(0);
});

test("Hedera Test Get Guardian Set Expiry", async () => {
  const client = getClient();
  expect(client).toBeDefined();

  const jsonFile = "../../../../../ethereum/build/contracts/Getters.json";
  const json = require(jsonFile);

  const contractId = ContractId.fromEvmAddress(0, 0, WH_ADDR);
  console.log(
    "Querying guardian set expiry, contract address: " +
      WH_ADDR +
      ", contractId " +
      contractId
  );

  const param = encodeFunctionCall("getGuardianSetExpiry", [], json.abi);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  console.log(
    "current guardian set expiry: " +
      decodeFunctionResult("getGuardianSetExpiry", result.bytes, json.abi)[0]
  );
});

test("Hedera Test Get Governance Contract", async () => {
  const client = getClient();
  expect(client).toBeDefined();

  const jsonFile = "../../../../../ethereum/build/contracts/Getters.json";
  const json = require(jsonFile);

  const contractId = ContractId.fromEvmAddress(0, 0, WH_ADDR);
  console.log(
    "Querying governance contract, contract address: " +
      WH_ADDR +
      ", contractId " +
      contractId
  );

  const param = encodeFunctionCall("governanceContract", [], json.abi);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  console.log("result", result);
  const govContResults = decodeFunctionResult(
    "governanceContract",
    result.bytes,
    json.abi
  );
  console.log("govContResults", govContResults);
  console.log("governance contract: ", govContResults[0]);
  expect(govContResults[0]).toBe(
    "0x0000000000000000000000000000000000000000000000000000000000000004"
  );
});

test("Hedera Test Get Guardian Set", async () => {
  const client = getClient();
  expect(client).toBeDefined();

  const jsonFile = "../../../../../ethereum/build/contracts/Getters.json";
  const json = require(jsonFile);

  const contractId = ContractId.fromEvmAddress(0, 0, WH_ADDR);

  const param = encodeFunctionCall("getCurrentGuardianSetIndex", [], json.abi);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  const decodedArray = decodeFunctionResult(
    "getCurrentGuardianSetIndex",
    result.bytes,
    json.abi
  )[0];
  console.log("result", decodedArray);
  const gsidx = decodedArray[0];
  console.log("current guardian set index: " + gsidx);

  console.log(
    "Querying guardian set for index " +
      gsidx +
      ", contract address: " +
      WH_ADDR +
      ", contractId " +
      contractId
  );

  const gsParam = encodeFunctionCall(
    "getGuardianSet",
    [parseInt(gsidx)],
    json.abi
  );

  // console.log("Making call, param: %o", gsParam);
  const gsResult = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(gsParam)
    .execute(client);

  console.log("Back from call", gsResult);

  const decodedGuardianSet = decodeFunctionResult(
    "getGuardianSet",
    gsResult.bytes,
    json.abi
  );
  console.log("guardian set: ", decodedGuardianSet);
  const decocdedInitSigner = decodedGuardianSet[0][0];
  // console.log("decode", decocdedInitSigner);
  expect(decocdedInitSigner[0]).toBe(INIT_SIGNERS[0]);
});

test("Hedera Test Token Bridge Chain ID", async () => {
  const client = getClient();
  expect(client).toBeDefined();

  const jsonFile = "../../../../../ethereum/build/contracts/Bridge.json";
  const json = require(jsonFile);

  const contractId = ContractId.fromEvmAddress(0, 0, TB_ADDR);
  console.log(
    "Querying chain ID, contract address: " +
      TB_ADDR +
      ", contractId " +
      contractId
  );
  const param = encodeFunctionCall("chainId", [], json.abi);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  console.log("result", result);
  const chainResult = decodeFunctionResult("chainId", result.bytes, json.abi);
  console.log("chainResult", chainResult[0]);
  expect(parseInt(chainResult[0])).toBe(CHAIN_ID_HEDERA);
});

test("Hedera Test Token Bridge Token Implementation Query", async () => {
  const client = getClient();
  expect(client).toBeDefined();

  const jsonFile = "../../../../../ethereum/build/contracts/Bridge.json";
  const json = require(jsonFile);

  const contractId = ContractId.fromEvmAddress(0, 0, TB_ADDR);
  console.log(
    "Querying token implementation, contract address: " +
      TB_ADDR +
      ", contractId " +
      contractId
  );
  const param = encodeFunctionCall("tokenImplementation", [], json.abi);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  console.log("result", result);
  const tokenResult = decodeFunctionResult(
    "tokenImplementation",
    result.bytes,
    json.abi
  );
  console.log("token implementation result:", tokenResult[0]);
  expect(tokenResult[0].toLowerCase()).toBe(TOKEN_IMPL.toLowerCase());
});

test.skip("Hedera Test Token Bridge Token IsWrappedAsset", async () => {
  const ASSET = "0x0000000000000000000000000000000002dc2a08";
  const client = getClient();

  const jsonFile = "../../../../../ethereum/build/contracts/Bridge.json";
  const json = require(jsonFile);

  const contractId = ContractId.fromEvmAddress(0, 0, TB_ADDR);
  console.log(
    "Querying token isWrappedAsset, contract address: " +
      TB_ADDR +
      ", contractId " +
      contractId
  );
  let param = encodeFunctionCall("isWrappedAsset", [ASSET], json.abi);

  let result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  console.log("result", result);
  const decodedResult = decodeFunctionResult(
    "isWrappedAsset",
    result.bytes,
    json.abi
  );
  console.log("isWrappedAsset result:", decodedResult[0]);
  expect(decodedResult[0]).toBe(false);

  console.log("Attempting wrappedAsset...");
  const uusd =
    "0x0100000000000000000000000000000000000000000000000000000075757364";
  param = encodeFunctionCall("wrappedAsset", ["27", ASSET], json.abi);
  console.log("Calling the contract with param:", param);

  result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

  console.log("result2", result);
  expect(result._createResult).toBeTruthy();
  console.log(
    "wrapped asset: " +
      decodeFunctionResult("wrappedAsset", result.bytes, json.abi)[0]
  );
});

test("Hedera Test Token Bridge Attest Token", async () => {
  const asset: string = process.env.TOKEN_ID || "";
  const tokenId: TokenId = TokenId.fromString(asset);
  const client = getClient();
  const nonce: number = Math.trunc(Math.random() * 100000);
  const nonceStr: string = nonce.toString(10);

  const jsonFile = "../../../../../ethereum/build/contracts/Bridge.json";
  const json = require(jsonFile);

  const TB_ADDR = "0000000000000000000000000000000002dc2a0e";
  const contractId = ContractId.fromEvmAddress(0, 0, TB_ADDR);
  const nativeAsset = tryHexToNativeAssetString(
    textToHexString(asset),
    CHAIN_ID_HEDERA
  );
  // ).substring(2);
  const tokenInSolidity = tokenId.toSolidityAddress();
  console.log("tokenId:", tokenInSolidity);
  console.log("nativeAsset:", nativeAsset, "nonce:", nonce);
  // const param = encodeFunctionCall(
  //   "attestToken",
  //   ["0x" + tokenInSolidity, nonceStr],
  //   json.abi
  // );
  // console.log("param", param);

  const cfp = new ContractFunctionParameters();
  cfp.addAddress(tokenInSolidity);
  cfp.addUint32(nonce);

  console.log("Attempting to get result...");
  const command = new ContractExecuteTransaction()
    .setContractId(contractId)
    .setGas(1_000_000)
    .setFunction("attestToken", cfp);
  console.log("command:", command);

  const txResponse = await command.execute(client);
  console.log("txResponse:", txResponse);
  console.log("transactionId:", txResponse.transactionId.toString());
  // Check the transactionId at https://hashscan.io/#/testnet/transactionsById/0.0.34399286-1662652168-920954953

  //Request the receipt of the transaction
  const receipt = await txResponse.getReceipt(client);
  console.log("receipt:", receipt);

  //Get the transaction consensus status
  const transactionStatus = receipt.status;
  console.log("The transaction consensus status is " + transactionStatus);
  expect(transactionStatus).toBe(Status.Success);
});

test.skip("Hedera Test Token Bridge Attest Token #2", async () => {
  const asset = process.env.TOKEN_ID || "";
  const client = getClient();
  const nonce = Math.trunc(Math.random() * 100000);
});

test.skip("Hedera Test Create New Account", async () => {
  const client = getClient();

  //Create new keys
  const newAccountPrivateKey = await PrivateKey.generateED25519();
  const newAccountPublicKey = newAccountPrivateKey.publicKey;

  //Create a new account with 1,000 tinybar starting balance
  const newAccount = await new AccountCreateTransaction()
    .setKey(newAccountPublicKey)
    .setInitialBalance(Hbar.fromTinybars(1000))
    .execute(client);

  // Get the new account ID
  const getReceipt = await newAccount.getReceipt(client);
  const newAccountId = getReceipt.accountId;

  console.log("The new account ID is: " + newAccountId);
  if (!newAccountId) {
    throw new Error("Bad newAccountId");
  }

  //Verify the account balance
  const accountBalance = await new AccountBalanceQuery()
    .setAccountId(newAccountId)
    .execute(client);

  console.log("accountBalance:", accountBalance);

  console.log(
    "The new account balance is: " +
      accountBalance.hbars.toTinybars() +
      " tinybar."
  );
});

test.skip("Hedera Test Create New Token", async () => {
  const client = getClient();
  const supplyKey = PrivateKey.generate();
  const treasuryId = AccountId.fromString(OPERATOR_ID);
  const treasuryKey = PrivateKey.fromString(OPERATOR_KEY);
  //CREATE FUNGIBLE TOKEN (STABLECOIN)
  let tokenCreateTx = await new TokenCreateTransaction()
    .setTokenName("USD Bar")
    .setTokenSymbol("USDB")
    .setTokenType(TokenType.FungibleCommon)
    .setDecimals(2)
    .setInitialSupply(10000)
    .setTreasuryAccountId(treasuryId)
    .setSupplyType(TokenSupplyType.Infinite)
    .setSupplyKey(supplyKey)
    .freezeWith(client);

  let tokenCreateSign = await tokenCreateTx.sign(treasuryKey);
  let tokenCreateSubmit = await tokenCreateSign.execute(client);
  let tokenCreateRx = await tokenCreateSubmit.getReceipt(client);
  let tokenId = tokenCreateRx.tokenId;
  console.log(`- Created token with ID: ${tokenId} \n`);
  if (!tokenId) {
    throw new Error("Bad tokenId");
  }

  //Verify the account balance
  const accountBalance = await new AccountBalanceQuery()
    .setAccountId(treasuryId)
    .execute(client);

  console.log("accountBalance:", accountBalance);
  // //TOKEN ASSOCIATION WITH ALICE's ACCOUNT
  // let associateTx = await new TokenAssociateTransaction()
  //   .setAccountId(OPERATOR_ID)
  //   .setTokenIds([tokenId])
  //   .freezeWith(client)
  //   .sign(treasuryKey);
  // let associateTxSubmit = await associateTx.execute(client);
  // let associateRx = await associateTxSubmit.getReceipt(client);
  // console.log("associateRx", associateRx);
  // console.log(`- Token association with account: ${associateRx.status} \n`);
});

test.skip("Hedera Test Conversions", async () => {
  let asset = "hbar";
  let hexStr = textToHexString(asset);
  let nativeAsset = tryHexToNativeAssetString(
    hexStr,
    CHAIN_ID_HEDERA
  ).substring(2);
  console.log("Asset:", asset, ", hex:", hexStr, ", native:", nativeAsset);
  let hexBackStr = tryNativeToHexString("0x" + nativeAsset, "hedera");
  asset = process.env.TOKEN_ID || "";
  hexStr = textToHexString(asset);
  nativeAsset = tryHexToNativeAssetString(hexStr, CHAIN_ID_HEDERA).substring(2);
  console.log("Asset:", asset, ", hex:", hexStr, ", native:", nativeAsset);
  hexBackStr = tryNativeToHexString("0x" + nativeAsset, "hedera");
});
