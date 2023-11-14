import {
  afterAll,
  beforeAll,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import Web3, { ETH_DATA_FORMAT } from "web3";
import axios from "axios";
import {
  EthCallData,
  EthCallQueryRequest,
  EthCallByTimestampQueryRequest,
  EthCallWithFinalityQueryRequest,
  PerChainQueryRequest,
  QueryRequest,
  sign,
} from "..";

jest.setTimeout(125000);

const CI = process.env.CI;
const ENV = "DEVNET";
const ETH_NODE_URL = CI ? "ws://eth-devnet:8545" : "ws://localhost:8545";

const SERVER_URL = CI ? "http://query-server:" : "http://localhost:";
const CCQ_SERVER_URL = SERVER_URL + "6069/v1";
const QUERY_URL = CCQ_SERVER_URL + "/query";
const HEALTH_URL = SERVER_URL + "6068/health";
const PRIVATE_KEY =
  "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";
const WETH_ADDRESS = "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E";

let web3: Web3;

beforeAll(() => {
  web3 = new Web3(ETH_NODE_URL);
});

afterAll(() => {
  web3.provider?.disconnect();
});

function createTestEthCallData(
  to: string,
  name: string,
  outputType: string
): EthCallData {
  return {
    to,
    data: web3.eth.abi.encodeFunctionCall(
      {
        constant: true,
        inputs: [],
        name,
        outputs: [{ name, type: outputType }],
        payable: false,
        stateMutability: "view",
        type: "function",
      },
      []
    ),
  };
}

async function getEthCallByTimestampArgs(): Promise<[bigint, bigint, bigint]> {
  let followingBlockNumber = BigInt(
    await web3.eth.getBlockNumber(ETH_DATA_FORMAT)
  );
  let targetBlockNumber = BigInt(0);
  let targetBlockTime = BigInt(0);
  while (targetBlockNumber === BigInt(0)) {
    let followingBlock = await web3.eth.getBlock(followingBlockNumber);
    while (Number(followingBlock.number) <= 0) {
      await sleep(1000);
      followingBlock = await web3.eth.getBlock(followingBlock.number);
      followingBlockNumber = followingBlock.number;
    }
    const targetBlock = await web3.eth.getBlock(
      (Number(followingBlockNumber) - 1).toString()
    );
    if (targetBlock.timestamp < followingBlock.timestamp) {
      targetBlockTime = targetBlock.timestamp * BigInt(1000000);
      targetBlockNumber = targetBlock.number;
    } else {
      followingBlockNumber = targetBlockNumber;
    }
  }
  return [targetBlockTime, targetBlockNumber, followingBlockNumber];
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

describe("eth call", () => {
  test("serialize request", () => {
    const toAddress = "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270";
    const nameCallData = createTestEthCallData(toAddress, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      toAddress,
      "totalSupply",
      "uint256"
    );
    const ethCall = new EthCallQueryRequest("0x28d9630", [
      nameCallData,
      totalSupplyCallData,
    ]);
    const chainId = 5;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    expect(Buffer.from(serialized).toString("hex")).toEqual(
      "0100000001010005010000004600000009307832386439363330020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd"
    );
  });
  test("successful query", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const blockNumber = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const ethCall = new EthCallQueryRequest(blockNumber, [
      nameCallData,
      totalSupplyCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    const response = await axios.put(
      QUERY_URL,
      {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      },
      { headers: { "X-API-Key": "my_secret_key" } }
    );
    expect(response.status).toBe(200);
  });
  // TODO: This test works in Goerli testnet but not devnet. Try it again after PR #3395 lands.
  test.skip("get block by hash should work", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const blockNumber = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const block = await web3.eth.getBlock(BigInt(blockNumber));
    if (block.hash != undefined) {
      const ethCall = new EthCallQueryRequest(block.hash?.toString(), [
        nameCallData,
        totalSupplyCallData,
      ]);
      const chainId = 2;
      const ethQuery = new PerChainQueryRequest(chainId, ethCall);
      const nonce = 1;
      const request = new QueryRequest(nonce, [ethQuery]);
      const serialized = request.serialize();
      const digest = QueryRequest.digest(ENV, serialized);
      const signature = sign(PRIVATE_KEY, digest);
      const response = await axios.put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key" } }
      );
      expect(response.status).toBe(200);
    }
  });
  test("missing api-key should fail", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const blockNumber = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const ethCall = new EthCallQueryRequest(blockNumber, [
      nameCallData,
      totalSupplyCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    let err = false;
    await axios
      .put(QUERY_URL, {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      })
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(401);
        expect(error.response.data).toBe("api key is missing\n");
      });
    expect(err).toBe(true);
  });
  test("invalid api-key should fail", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const blockNumber = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const ethCall = new EthCallQueryRequest(blockNumber, [
      nameCallData,
      totalSupplyCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    let err = false;
    await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "some_junk" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(403);
        expect(error.response.data).toBe("invalid api key\n");
      });
    expect(err).toBe(true);
  });
  test("unauthorized call should fail", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const blockNumber = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const ethCall = new EthCallQueryRequest(blockNumber, [
      nameCallData,
      totalSupplyCallData, // API key "my_secret_key_2" is not authorized to do total supply.
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    let err = false;
    await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key_2" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toBe(
          `call "ethCall:2:000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e:18160ddd" not authorized\n`
        );
      });
    expect(err).toBe(true);
  });
  test("unsigned query should fail if not allowed", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const blockNumber = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const ethCall = new EthCallQueryRequest(blockNumber, [
      nameCallData,
      totalSupplyCallData,
    ]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const signature = "";
    let err = false;
    await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toBe(`request not signed\n`);
      });
    expect(err).toBe(true);
  });
  test("unsigned query should succeed if allowed", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const blockNumber = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const ethCall = new EthCallQueryRequest(blockNumber, [nameCallData]);
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const signature = "";
    const response = await axios.put(
      QUERY_URL,
      {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      },
      { headers: { "X-API-Key": "my_secret_key_2" } } // This API key allows unsigned queries.
    );
    expect(response.status).toBe(200);
  });
  test("health check", async () => {
    const response = await axios.get(HEALTH_URL);
    expect(response.status).toBe(200);
  });
  test("payload too large should fail", async () => {
    const serialized = new Uint8Array(6000000); // Buffer should be larger than MAX_BODY_SIZE in node/cmd/ccq/http.go.
    const signature = "";
    let err = false;
    await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toBe(`http: request body too large\n`);
      });
    expect(err).toBe(true);
  });
  test("serialize eth_call_by_timestamp request", () => {
    const toAddress = "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270";
    const nameCallData = createTestEthCallData(toAddress, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      toAddress,
      "totalSupply",
      "uint256"
    );
    const ethCall = new EthCallByTimestampQueryRequest(
      BigInt(1697216322000000),
      "0x28d9630",
      "0x28d9631",
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 5;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    expect(Buffer.from(serialized).toString("hex")).toEqual(
      "0100000001010005020000005b0006079bf7fad4800000000930783238643936333000000009307832386439363331020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd"
    );
  });
  test("successful eth_call_by_timestamp query with block hints", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const [targetBlockTime, targetBlockNumber, followingBlockNumber] =
      await getEthCallByTimestampArgs();
    const ethCall = new EthCallByTimestampQueryRequest(
      targetBlockTime,
      targetBlockNumber.toString(16),
      followingBlockNumber.toString(16),
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    const response = await axios.put(
      QUERY_URL,
      {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      },
      { headers: { "X-API-Key": "my_secret_key" } }
    );
    expect(response.status).toBe(200);
  });
  test("successful eth_call_by_timestamp query without block hints", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const [targetBlockTime, targetBlockNumber, followingBlockNumber] =
      await getEthCallByTimestampArgs();
    const ethCall = new EthCallByTimestampQueryRequest(
      targetBlockTime + BigInt(5000),
      "",
      "",
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    const response = await axios.put(
      QUERY_URL,
      {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      },
      { headers: { "X-API-Key": "my_secret_key" } }
    );
    expect(response.status).toBe(200);
  });
  test("eth_call_by_timestamp query without target timestamp", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const followingBlockNum = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const followingBlock = await web3.eth.getBlock(BigInt(followingBlockNum));
    const targetBlock = await web3.eth.getBlock(
      (Number(followingBlockNum) - 1).toString()
    );
    const ethCall = new EthCallByTimestampQueryRequest(
      BigInt(0),
      targetBlock.number.toString(16),
      followingBlock.number.toString(16),
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    let err = false;
    const response = await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toBe(
          `failed to validate request: failed to validate per chain query 0: chain specific query is invalid: target timestamp may not be zero\n`
        );
      });
    expect(err).toBe(true);
  });
  test("eth_call_by_timestamp query with following hint but not target hint should fail", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const followingBlockNum = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const followingBlock = await web3.eth.getBlock(BigInt(followingBlockNum));
    const targetBlock = await web3.eth.getBlock(
      (Number(followingBlockNum) - 1).toString()
    );
    const targetBlockTime = targetBlock.timestamp * BigInt(1000000);
    const ethCall = new EthCallByTimestampQueryRequest(
      targetBlockTime,
      "",
      followingBlock.number.toString(16),
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    let err = false;
    const response = await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toBe(
          `failed to validate request: failed to validate per chain query 0: chain specific query is invalid: if either the target or following block id is unset, they both must be unset\n`
        );
      });
    expect(err).toBe(true);
  });
  test("eth_call_by_timestamp query with target hint but not following hint should fail", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const followingBlockNum = await web3.eth.getBlockNumber(ETH_DATA_FORMAT);
    const targetBlock = await web3.eth.getBlock(
      (Number(followingBlockNum) - 1).toString()
    );
    const targetBlockTime = targetBlock.timestamp * BigInt(1000000);
    const ethCall = new EthCallByTimestampQueryRequest(
      targetBlockTime,
      targetBlock.number.toString(16),
      "",
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    let err = false;
    const response = await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toBe(
          `failed to validate request: failed to validate per chain query 0: chain specific query is invalid: if either the target or following block id is unset, they both must be unset\n`
        );
      });
    expect(err).toBe(true);
  });
  test("serialize eth_call_with_finality request", () => {
    const toAddress = "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270";
    const nameCallData = createTestEthCallData(toAddress, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      toAddress,
      "totalSupply",
      "uint256"
    );
    const ethCall = new EthCallWithFinalityQueryRequest(
      "0x28d9630",
      "finalized",
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 5;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    expect(Buffer.from(serialized).toString("hex")).toEqual(
      "01000000010100050300000053000000093078323864393633300000000966696e616c697a6564020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd"
    );
  });
  test("successful eth_call_with_finality query", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    // Jump into the future a bit so the watcher has to wait for finality.
    const blockNumber =
      Number(await web3.eth.getBlockNumber(ETH_DATA_FORMAT)) + 10;
    const ethCall = new EthCallWithFinalityQueryRequest(
      blockNumber.toString(16),
      "finalized",
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    const response = await axios.put(
      QUERY_URL,
      {
        signature,
        bytes: Buffer.from(serialized).toString("hex"),
      },
      { headers: { "X-API-Key": "my_secret_key" } }
    );
    expect(response.status).toBe(200);
  });
  test("eth_call_with_finality query without finality should fail", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const ethCall = new EthCallWithFinalityQueryRequest(
      "0x28d9630",
      "" as any,
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    let err = false;
    const response = await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toBe(
          `failed to validate request: failed to validate per chain query 0: chain specific query is invalid: finality is required\n`
        );
      });
    expect(err).toBe(true);
  });
  test("eth_call_with_finality query with bad finality should fail", async () => {
    const nameCallData = createTestEthCallData(WETH_ADDRESS, "name", "string");
    const totalSupplyCallData = createTestEthCallData(
      WETH_ADDRESS,
      "totalSupply",
      "uint256"
    );
    const ethCall = new EthCallWithFinalityQueryRequest(
      "0x28d9630",
      "HelloWorld" as any,
      [nameCallData, totalSupplyCallData]
    );
    const chainId = 2;
    const ethQuery = new PerChainQueryRequest(chainId, ethCall);
    const nonce = 1;
    const request = new QueryRequest(nonce, [ethQuery]);
    const serialized = request.serialize();
    const digest = QueryRequest.digest(ENV, serialized);
    const signature = sign(PRIVATE_KEY, digest);
    let err = false;
    const response = await axios
      .put(
        QUERY_URL,
        {
          signature,
          bytes: Buffer.from(serialized).toString("hex"),
        },
        { headers: { "X-API-Key": "my_secret_key" } }
      )
      .catch(function (error) {
        err = true;
        expect(error.response.status).toBe(400);
        expect(error.response.data).toBe(
          `failed to validate request: failed to validate per chain query 0: chain specific query is invalid: finality must be "finalized" or "safe", is "HelloWorld"\n`
        );
      });
    expect(err).toBe(true);
  });
});
