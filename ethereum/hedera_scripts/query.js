require("dotenv").config({ path: ".env" });
const {
  AccountId,
  PrivateKey,
  Client,
  ContractId,
  ContractCallQuery,
  ContractExecuteTransaction,
  ContractFunctionParameters,
  ContractInfoQuery,
  Hbar,
  TokenId,
} = require("@hashgraph/sdk");

// Configure accounts and client
const operatorId = AccountId.fromString(process.env.OPERATOR_ID)
const operatorKey = PrivateKey.fromString(process.env.OPERATOR_PVKEY)

async function queryContractInfo(contractName, contractAddress) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Querying " + contractName + " contract info, contract address: " + contractAddress + ", contractId " + contractId)

  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  const query = new ContractInfoQuery().setContractId(contractId);

  //Sign the query with the client operator private key and submit to a Hedera network
  const info = await query.execute(client);
  console.log("contract info for " + contractName + ": %o", info);
}

async function queryChainId(contractName, contractAddress, jsonFile) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Querying " + contractName + " chainId, contract address: " + contractAddress + ", contractId " + contractId)

  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunction("chainId")
    .execute(client);

  console.log("chainId: " + result.getUint32(0))
}

//////////////////////////////////// Wormhole specific stuff

async function queryWormholeStuff(contractAddress) {
  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  const gsidx = await getCurrentGuardianSetIndex(client, contractAddress)
  await getGuardianSetExpiry(client, contractAddress)
  await getGovernanceContract(client, contractAddress)

  // Not sure why this is failing:
  await getGuardianSet(client, contractAddress, gsidx)
}

async function getCurrentGuardianSetIndex(client, contractAddress) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Querying guardian set index, contract address: " + contractAddress + ", contractId " + contractId)

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunction("getCurrentGuardianSetIndex")
    .execute(client);

  const gsidx = result.getUint32(0)
  console.log("current guardian set index: " + gsidx)
  return gsidx
}

async function getGuardianSetExpiry(client, contractAddress) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Querying guardian set expiry, contract address: " + contractAddress + ", contractId " + contractId)

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunction("getGuardianSetExpiry")
    .execute(client);

  const gsexpy = result.getUint32(0)
  console.log("current guardian set expiry: " + gsexpy)
}

// This fails due to gas.
async function getGuardianSet(client, contractAddress, gsidx) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Querying guardian set for index " + gsidx + ", contract address: " + contractAddress + ", contractId " + contractId)

  console.log("creating query")
  const query = await new ContractCallQuery()
    .setMaxQueryPayment(new Hbar(100))
    .setQueryPayment(new Hbar(20))
    .setContractId(contractId)
    .setGas(new Hbar(100))
    .setFunction("getGuardianSet", new ContractFunctionParameters().addUint32(gsidx))    

  // const cost = await query.getCost(client)
  // console.log("query would cost: " + cost)

  console.log("executing query")
  const result = await query.execute(client);

  console.log("Back from query")
  console.log("getGuardianSet[" + gsidx + "]: " + result)
}

async function getGovernanceContract(client, contractAddress) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Querying governance contract, contract address: " + contractAddress + ", contractId " + contractId)

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunction("governanceContract")
    .execute(client);

  console.log("governanceContract: " + result.getBytes32(0).toString("hex"))
}

//////////////////////////////////// Bridge specific stuff

async function queryBridgeContract(contractName, contractAddress, chainId) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Querying bridge contract from " + contractName + ", contract address: " + contractAddress + ", contractId " + contractId + ", chainId " + chainId)

  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunction("bridgeContracts", new ContractFunctionParameters().addUint16(chainId))
    .execute(client);

    console.log("bridgeContract[" + chainId + "]: " + result.getBytes32(0).toString("hex"))
}

async function registerChain(contractName, contractAddress, vaa) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Registering chain on " + contractName + ", contract address: " + contractAddress + ", contractId " + contractId)

  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  client.s

  if (vaa.search("0x") === 0) {
    vaa = vaa.substring(2)
  }

  let array = Array.from(Uint8Array.from(Buffer.from(vaa, 'hex')))
  console.log("BOINK: %o", array)

  const txResponse = await new ContractExecuteTransaction()
    .setContractId(contractId)
    .setGas(1000000)
    .setFunction("registerChain", new ContractFunctionParameters().addUint8Array(array))
    .execute(client);

  //Request the receipt of the transaction
  const receipt = await txResponse.getReceipt(client);

  console.log("registerChain txResponse: %o", txResponse)  
  console.log("registerChain txHash: %o", txResponse.transactionHash)
  
  console.log("registerChain receipt: %o", receipt)  

  //Get the transaction consensus status
  const transactionStatus = receipt.status;

  console.log("The transaction consensus status is " + transactionStatus);    
}

async function createWrapped(contractName, contractAddress, vaa, jsonFile) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Creating wrapped asset on " + contractName + ", contract address: " + contractAddress + ", contractId " + contractId)

  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  const json = require(jsonFile);
  const param = encodeFunctionCall("createWrapped", [ vaa ], json.abi);

  const txResponse = await new ContractExecuteTransaction()
    .setContractId(contractId)
    .setGas(1000000)
    .setFunctionParameters(param)
    .execute(client);


  //Request the receipt of the transaction
  const receipt = await txResponse.getReceipt(client);

  console.log("createWrapped txResponse: %o", txResponse)  
  console.log("createWrapped receipt: %o", receipt)  

  //Get the transaction consensus status
  const transactionStatus = receipt.status;

  console.log("The transaction consensus status is " + transactionStatus);    
}

async function queryWrappedAsset(contractName, contractAddress) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Getting wrapped asset on " + contractName + ", contract address: " + contractAddress + ", contractId " + contractId)

  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  const addr = new Uint8Array(Buffer.from("0100000000000000000000000000000000000000000000000000000075757364", 'hex'))

  const result = await new ContractCallQuery()
  .setContractId(contractId)
  .setGas(100000)
  .setFunction("wrappedAsset", new ContractFunctionParameters().addUint16(3).addBytes32(addr))
  .execute(client);

  console.log("wrapped asset: " + result.getBytes32(0).toString("hex"))
}

// This fails due to gas.
async function queryToken(contractName, tokenAddress) {
  const contractId = ContractId.fromEvmAddress(0, 0, tokenAddress.substring(2));
  console.log("Getting token info on " + contractName + ", token address: " + tokenAddress + ", contractId " + contractId)

  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  {
    const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunction("symbol")
    .execute(client);

    console.log("symbol: " + result.getString(0))
  }

  {
    const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunction("chainId")
    .execute(client);

    console.log("origin chain id: " + decodeFunctionResult("chainId", result.bytes, json.abi)[0])  
  }
  
  {
    const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunction("nativeContract")
    .execute(client);

    console.log("native contract: " + decodeFunctionResult("nativeContract", result.bytes, json.abi)[0])  
  }
    
  {
    const walletAddr = "0x00000000000000000000000000000000020ce436"
    const param = encodeFunctionCall("balanceOf", [ walletAddr ], json.abi);

    const result = await new ContractCallQuery()
    .setContractId(contractId)
    .setGas(100000)
    .setFunctionParameters(param)
    .execute(client);

    console.log("balance: " + decodeFunctionResult("balanceOf", result.bytes, json.abi)[0])  
  }
}

async function completeTransfer(contractName, contractAddress, vaa, jsonFile) {
  const contractId = ContractId.fromEvmAddress(0, 0, contractAddress.substring(2));
  console.log("Registering chain on " + contractName + ", contract address: " + contractAddress + ", contractId " + contractId)

  const client = Client.forTestnet().setOperator(operatorId, operatorKey);

  const json = require(jsonFile);
  const param = encodeFunctionCall("completeTransfer", [ vaa ], json.abi);

  const txResponse = await new ContractExecuteTransaction()
    .setContractId(contractId)
    .setGas(1000000)
    .setFunctionParameters(param)
    .execute(client);


  //Request the receipt of the transaction
  const receipt = await txResponse.getReceipt(client);

  console.log("completeTransfer txResponse: %o", txResponse)  
  console.log("completeTransfer receipt: %o", receipt)  

  //Get the transaction consensus status
  const transactionStatus = receipt.status;

  console.log("The transaction consensus status is " + transactionStatus);
}

async function main() {
  await queryContractInfo("Wormhole", "0x00000000000000000000000000000000020cea83")
  await queryContractInfo("TokenBridge", "0x00000000000000000000000000000000020ceb06")
  await queryContractInfo("NFTBridge", "0x00000000000000000000000000000000020ceb0e")
  // await queryChainId("Wormhole", "0x00000000000000000000000000000000020cea83")
  // await queryChainId("TokenBridge", "0x00000000000000000000000000000000020ceb06")
  // await queryChainId("NFTBridge", "0x00000000000000000000000000000000020ceb0e")

  // const VAA1 = "0x01000000000100c0c64027377f27bb3ef676f2fd9c63a30a6d1bf904851656cad635601425191017308e3d0e605364c3a1a5305b722fd150bde1f869b74a8cbb686774c41a7c880100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000051a364b00000000000000000000000000000000000000000000546f6b656e42726964676501000000013b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98"
  // const VAA2 = "0x01000000000100b3644d45f6e442c913719ab546f4052c296bbf8c0541e41170b5f5c525cc52181ee42a40f34fa01f6b9256375e31de4e5c52e80cb7afebffb3e559fa70f6e5e400000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000369a27e00000000000000000000000000000000000000000000546f6b656e4272696467650100000002000000000000000000000000f890982f9310df57d00f659cf4fd87e65aded8d7"
  // const VAA3 = "0x01000000000100f98926ea6a4e603985331c93ec48a3d21c4a8214e9e60013d94f9f29d0af19381d63124af52c21ce704fe64eda1c1ef8e1224db37342b6f70479f5f2a6e9715e000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000001ec910d00000000000000000000000000000000000000000000546f6b656e42726964676501000000030000000000000000000000000c32d68d8f22613f6b9511872dad35a59bfdf7f0"
  // const VAA4 = "0x01000000000100e41b7eb99f0544ffa5041916edce298d2fc2a931f106b9dedb31ea02314332a24d8412e6a40f1147796f9e76dfd84639d5e92c81f4d6da91a8ce37d4747915c1000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000000e3ba2d00000000000000000000000000000000000000000000546f6b656e42726964676501000000040000000000000000000000009dcf9d205c9de35334d646bee44b2d2859712a09"
  // const VAA10 = "0x010000000001007bc5bdc2253a6deec065a6e573f7bdd363383b91210054d8ce7558d9b8185a60157370f56f35d03fc434ecf90fc652cf31b9b4a2a15eb43c5ad8a664356ef81e00000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000351c06c00000000000000000000000000000000000000000000546f6b656e427269646765010000000a000000000000000000000000599cea2204b4faecd584ab1f2b6aca137a0afbe8"
  // const VAA11 = "0x01000000000100c01c50054de5f481bb5cd8b4e5d906d86c7307ba969a0f9f042e410b859528387040360954dc4fd4834d4f68dfea9067f3da3bb4af127f61e3ae04f7d216fc300100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000026bcc5b00000000000000000000000000000000000000000000546f6b656e427269646765010000000b000000000000000000000000d11de1f930ea1f7dd0290fe3a2e35b9c91aefb37"
  // const VAA12 = "0x01000000000100595f3b02b450ea71724c09a4b1a071f67eafb1471e0bd0d03d506e11ceedc81d788dd57dc81927cb74970ad6389e369d0589682d176eff033fe140dae1035155000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002a49e7a00000000000000000000000000000000000000000000546f6b656e427269646765010000000c000000000000000000000000eba00cbe08992edd08ed7793e07ad6063c807004"
  // const VAA13 = "0x0100000000010018f4c45a9e8e0b6767d84bef689987dcdf86190500b92c82aa67df532a5719451b5a1897c342ae4fb851fc84c090624ab3070b485f54a07b9cf9e7645069c8cf010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004df8a7400000000000000000000000000000000000000000000546f6b656e427269646765010000000d000000000000000000000000c7a13be098720840dea132d860fdfa030884b09a"
  // const VAA14 = "0x010000000001008a8e90d7053e7f2056a637de9cf525e949a2a8834b9c3a8ac05b6106dd83756443493fcc75a88f4c6fcd9cf1121a7b5f51807f8a9e3e63cebdd47904f30de70d000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000001b0920b00000000000000000000000000000000000000000000546f6b656e427269646765010000000e00000000000000000000000005ca6037ec51f8b712ed2e6fa72219feae74e153"
  // const VAA16 = "0x010000000001000f89d8382c4e3cc006f47cacbdc3778dc4538735beb4b3b16f1298ae5d3343b76a5a7f219cf164cb3d825d835171657853a5a6737d01e34b3cd185f7359ae981010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000005c65ed800000000000000000000000000000000000000000000546f6b656e4272696467650100000010000000000000000000000000bc976d4b9d57e57c3ca52e1fd136c45ff7955a96"
  // const VAA17 = "0x01000000000100c3dfa36528d33e2c27986b354363584ef50b02bc4449b0ca1626568cc259e183334b3af5ead6d704548e820f0b8cfefd043310517edeabb059a23ec4b54971560000000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000053f6dc600000000000000000000000000000000000000000000546f6b656e4272696467650100000011000000000000000000000000d11de1f930ea1f7dd0290fe3a2e35b9c91aefb37"
  // const VAA18 = "0x0100000000010012e2cdcc83e46393100a4db2e4ffd1d01fc9202fcf23355f45789f85a1d8329d6da5cfbe6571006e5f87f5ac38f2b153634f04fb7a115bdb066761c506a68416010000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002ef7f3d00000000000000000000000000000000000000000000546f6b656e4272696467650100000012c3d4c6c2bcba163de1defb7e8f505cdb40619eee4fa618678955e8790ae1448d"
  // const VAA10001 = "0x01000000000100d150e028e4253a9e9979ae8c816e45c0ad91aca59d894f10edc339531a4b92ce250f0f30b2d9473004eb0e5125542170c4c0d36a6832fc06e0d118c3ada5e8cb000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000002ad3fef00000000000000000000000000000000000000000000546f6b656e4272696467650100002711000000000000000000000000f174f9a837536c449321df1ca093bb96948d5386"
  // await registerChain("TokenBridge", "0x00000000000000000000000000000000020ceb06", VAA10)
  // for (let chainId = 1; (chainId <= 17); chainId++) {
  //   await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", chainId)
  // }
  // await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", 3)
  // await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", 10)
  // await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", 11)
  // await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", 12)
  // await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", 13)
  // await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", 14)
  // await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", 16)
  // await queryBridgeContract("TokenBridge", "0x00000000000000000000000000000000020ceb06", 17)
  // const CREATE_VAA = "0x01000000000100714c45a356c917767c33fa614ee96cfd1cc120eeb240f92b1dccf427efa0fb1166fd7461041d3888fa31482b9c3f1f206a6cd2bcf9b20ddbcacf97ab871ee34101627539c1000134cb00030000000000000000000000000c32d68d8f22613f6b9511872dad35a59bfdf7f0000000000000087e0002010000000000000000000000000000000000000000000000000000007575736400030655535400000000000000000000000000000000000000000000000000000000005553540000000000000000000000000000000000000000000000000000000000"
  // await createWrapped("TokenBridge", "0x00000000000000000000000000000000020ceb06", CREATE_VAA, "../build/contracts/Bridge.json")
  // await queryWrappedAsset("TokenBridge", "0x00000000000000000000000000000000020ceb06")
  // const COMPLETE_TRANSFER_VAA = "0x010000000001001657dfc7aa5f7891518838e9e9e836b97943bf9ca218c3a35e486bc94340bafc6432aa0bc06daaeaecdc7de86e522a65d6337295c880cf133898c98bb2aafb85006275507f00000c9600030000000000000000000000000c32d68d8f22613f6b9511872dad35a59bfdf7f00000000000000885000100000000000000000000000000000000000000000000000000000000000027100100000000000000000000000000000000000000000000000000000075757364000300000000000000000000000000000000000000000000000000000000020ce43600110000000000000000000000000000000000000000000000000000000000000000"
  //await completeTransfer("TokenBridge", "0x00000000000000000000000000000000020ceb06", COMPLETE_TRANSFER_VAA, "../build/contracts/Bridge.json")
  // await queryToken("TokenImplementation", "0xfa3383F9F111E78A2824903D5aE0016f6EFdC2F8")
  // await queryWormholeStuff("0x00000000000000000000000000000000020cea83")

  // console.log(TokenId.fromEvmAddress(0, 0, "0xfa3383F9F111E78A2824903D5aE0016f6EFdC2F8").toString())
  // console.log(TokenId.fromSolidityAddress("0xfa3383F9F111E78A2824903D5aE0016f6EFdC2F8").toString())

  // console.log("disconnecting web3")
  // await web3.eth.disconnect();

  console.log("All done.") 
}

main()

/*
Registering chain on TokenBridge, contract address: 0x00000000000000000000000000000000020ceb06, contractId 0.0.00000000000000000000000000000000020ceb06
registerChain txResponse: TransactionResponse {
  nodeId: AccountId {
    shard: Long { low: 0, high: 0, unsigned: false, [__isLong__]: true },
    realm: Long { low: 0, high: 0, unsigned: false, [__isLong__]: true },
    num: Long { low: 5, high: 0, unsigned: false, [__isLong__]: true },
    aliasKey: null,
    _checksum: null,
    [checksum]: [Getter]
  },

  // A random transaction ID looks like this: 0.0.34399286@1651776732.394128697. Before the @ is the accountId. After the @ is the timestamp.
  //0xfa0b871ea37ececc004ac2ebfd895628955e4da96b79821ef558d231e63d22d1041a208526543f34ad4671ee46e3b0bc
  transactionHash: <Buffer fa 0b 87 1e a3 7e ce cc 00 4a c2 eb fd 89 56 28 95 5e 4d a9 6b 79 82 1e f5 58 d2 31 e6 3d 22 d1 04 1a 20 85 26 54 3f 34 ad 46 71 ee 46 e3 b0 bc>,
  transactionId: TransactionId {
    accountId: AccountId {
      shard: Long { low: 0, high: 0, unsigned: false, [__isLong__]: true },
      realm: Long { low: 0, high: 0, unsigned: false, [__isLong__]: true },
      num: Long {
        low: 34399286,
        high: 0,
        unsigned: false,
        [__isLong__]: true
      },
      aliasKey: null,
      _checksum: null,
      [checksum]: [Getter]
    },
    validStart: Timestamp {
      seconds: Long {
        low: 1651777038,
        high: 0,
        unsigned: false,
        [__isLong__]: true
      },
      nanos: Long {
        low: 876614469,
        high: 0,
        unsigned: false,
        [__isLong__]: true
      }
    },
    scheduled: false,
    nonce: null
  }
}
registerChain receipt: TransactionReceipt {
  status: Status { _code: 22 },
  accountId: null,
  fileId: null,
  contractId: ContractId {
    shard: Long { low: 0, high: 0, unsigned: false, [__isLong__]: true },
    realm: Long { low: 0, high: 0, unsigned: false, [__isLong__]: true },
    num: Long {
      low: 34401030,
      high: 0,
      unsigned: false,
      [__isLong__]: true
    },
    evmAddress: null,
    _checksum: null,
    [checksum]: [Getter]
  },
  topicId: null,
  tokenId: null,
  scheduleId: null,
  exchangeRate: ExchangeRate {
    hbars: 30000,
    cents: 411265,
    expirationTime: 2022-05-05T19:00:00.000Z,
    exchangeRateInCents: 13.708833333333333
  },
  topicSequenceNumber: Long { low: 0, high: 0, unsigned: false, [__isLong__]: true },
  topicRunningHash: Uint8Array(0) [
    [BYTES_PER_ELEMENT]: 1,
    [length]: 0,
    [byteLength]: 0,
    [byteOffset]: 0,
    [buffer]: ArrayBuffer { byteLength: 0 }
  ],
  totalSupply: Long { low: 0, high: 0, unsigned: false, [__isLong__]: true },
  scheduledTransactionId: null,
  serials: [ [length]: 0 ],
  duplicates: [ [length]: 0 ],
  children: [ [length]: 0 ]
}
The transaction consensus status is 22 // Of all the stupidity, 22 is success.
*/
