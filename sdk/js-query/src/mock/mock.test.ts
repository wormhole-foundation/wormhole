import {
  afterAll,
  beforeAll,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import axios from "axios";
import base58 from "bs58";
import { eth } from "web3";
import {
  EthCallByTimestampQueryRequest,
  EthCallQueryRequest,
  EthCallWithFinalityQueryRequest,
  PerChainQueryRequest,
  QueryProxyMock,
  QueryProxyQueryResponse,
  QueryRequest,
  QueryResponse,
  SolanaAccountQueryRequest,
  SolanaAccountQueryResponse,
  SolanaPdaEntry,
  SolanaPdaQueryRequest,
  SolanaPdaQueryResponse,
} from "..";

jest.setTimeout(120000);

const SOLANA_NODE_URL = "http://localhost:8899";
const POLYGON_NODE_URL = "https://polygon-mumbai-bor.publicnode.com";
const ARBITRUM_NODE_URL = "https://arbitrum-goerli.publicnode.com";
const QUERY_URL = "https://testnet.ccq.vaa.dev/v1/query";

const SOL_PDAS: SolanaPdaEntry[] = [
  {
    programAddress: Uint8Array.from(
      base58.decode("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
    ), // Core Bridge address
    seeds: [
      new Uint8Array(Buffer.from("GuardianSet")),
      new Uint8Array(Buffer.alloc(4)),
    ], // Use index zero in tilt.
  },
];

let mock: QueryProxyMock;

beforeAll(() => {
  mock = new QueryProxyMock({
    1: SOLANA_NODE_URL,
    5: POLYGON_NODE_URL,
    23: ARBITRUM_NODE_URL,
  });
});

afterAll(() => {});

describe.skip("mocks match testnet", () => {
  test("EthCallQueryRequest mock matches testnet", async () => {
    const blockNumber = (
      await axios.post(POLYGON_NODE_URL, {
        jsonrpc: "2.0",
        id: 1,
        method: "eth_getBlockByNumber",
        params: ["latest", false],
      })
    ).data?.result?.number;
    expect(blockNumber).toBeTruthy();
    const polygonDemoContract = "0x130Db1B83d205562461eD0720B37f1FBC21Bf67F";
    const data = eth.abi.encodeFunctionSignature("getMyCounter()");
    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        5,
        new EthCallQueryRequest(blockNumber, [
          { to: polygonDemoContract, data },
        ])
      ),
    ]);
    const { bytes, signatures } = await mock.mock(query);
    // from CCQ Demo UI
    const signatureNotRequiredApiKey = "2d6c22c6-afae-4e54-b36d-5ba118da646a";
    const realResponse = (
      await axios.post<QueryProxyQueryResponse>(
        QUERY_URL,
        {
          bytes: Buffer.from(query.serialize()).toString("hex"),
        },
        { headers: { "X-API-Key": signatureNotRequiredApiKey } }
      )
    ).data;
    // the mock has an empty request signature, whereas the real service is signed
    // we'll empty out the sig to compare the bytes
    const realResponseWithEmptySignature = `${realResponse.bytes.substring(
      0,
      6
    )}${Buffer.from(new Array(65)).toString(
      "hex"
    )}${realResponse.bytes.substring(6 + 65 * 2)}`;
    expect(bytes).toEqual(realResponseWithEmptySignature);
    // similarly, we'll resign the bytes, to compare the signatures (only works with testnet key)
    // const serializedResponse = Buffer.from(realResponse.bytes, "hex");
    // const matchesReal = mock.sign(serializedResponse);
    // expect(matchesReal).toEqual(realResponse.signatures);
  });
  test("EthCallWithFinalityQueryRequest mock matches testnet", async () => {
    const blockNumber = (
      await axios.post(ARBITRUM_NODE_URL, {
        jsonrpc: "2.0",
        id: 1,
        method: "eth_getBlockByNumber",
        params: ["finalized", false],
      })
    ).data?.result?.number;
    expect(blockNumber).toBeTruthy();
    const arbitrumDemoContract = "0x6E36177f26A3C9cD2CE8DDF1b12904fe36deA47F";
    const data = eth.abi.encodeFunctionSignature("getMyCounter()");
    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        23,
        new EthCallWithFinalityQueryRequest(blockNumber, "finalized", [
          { to: arbitrumDemoContract, data },
        ])
      ),
    ]);
    const { bytes } = await mock.mock(query);
    // from CCQ Demo UI
    const signatureNotRequiredApiKey = "2d6c22c6-afae-4e54-b36d-5ba118da646a";
    const realResponse = (
      await axios.post<QueryProxyQueryResponse>(
        QUERY_URL,
        {
          bytes: Buffer.from(query.serialize()).toString("hex"),
        },
        { headers: { "X-API-Key": signatureNotRequiredApiKey } }
      )
    ).data;
    // the mock has an empty request signature, whereas the real service is signed
    // we'll empty out the sig to compare the bytes
    const realResponseWithEmptySignature = `${realResponse.bytes.substring(
      0,
      6
    )}${Buffer.from(new Array(65)).toString(
      "hex"
    )}${realResponse.bytes.substring(6 + 65 * 2)}`;
    expect(bytes).toEqual(realResponseWithEmptySignature);
    // similarly, we'll resign the bytes, to compare the signatures (only works with testnet key)
    // const serializedResponse = Buffer.from(realResponse.bytes, "hex");
    // const matchesReal = mock.sign(serializedResponse);
    // expect(matchesReal).toEqual(realResponse.signatures);
  });
  test("EthCallByTimestampQueryRequest mock matches testnet", async () => {
    const targetTimestamp =
      BigInt(Date.now() - 1000 * 30) * // thirty seconds ago
      BigInt(1000); // milliseconds to microseconds
    const arbitrumDemoContract = "0x6E36177f26A3C9cD2CE8DDF1b12904fe36deA47F";
    const data = eth.abi.encodeFunctionSignature("getMyCounter()");
    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        23,
        new EthCallByTimestampQueryRequest(targetTimestamp, "", "", [
          { to: arbitrumDemoContract, data },
        ])
      ),
    ]);
    const { bytes } = await mock.mock(query);
    // from CCQ Demo UI
    const signatureNotRequiredApiKey = "2d6c22c6-afae-4e54-b36d-5ba118da646a";
    const realResponse = (
      await axios.post<QueryProxyQueryResponse>(
        QUERY_URL,
        {
          bytes: Buffer.from(query.serialize()).toString("hex"),
        },
        { headers: { "X-API-Key": signatureNotRequiredApiKey } }
      )
    ).data;
    // the mock has an empty request signature, whereas the real service is signed
    // we'll empty out the sig to compare the bytes
    const realResponseWithEmptySignature = `${realResponse.bytes.substring(
      0,
      6
    )}${Buffer.from(new Array(65)).toString(
      "hex"
    )}${realResponse.bytes.substring(6 + 65 * 2)}`;
    expect(bytes).toEqual(realResponseWithEmptySignature);
  });
  test("EthCallByTimestampQueryRequest fails with non-adjacent blocks", async () => {
    expect.assertions(1);
    const targetTimestamp =
      BigInt(Date.now() - 1000 * 60 * 1) * // one minute ago
      BigInt(1000); // milliseconds to microseconds
    const blockNumber = (
      await axios.post(ARBITRUM_NODE_URL, {
        jsonrpc: "2.0",
        id: 1,
        method: "eth_getBlockByNumber",
        params: ["finalized", false],
      })
    ).data?.result?.number;
    const arbitrumDemoContract = "0x6E36177f26A3C9cD2CE8DDF1b12904fe36deA47F";
    const data = eth.abi.encodeFunctionSignature("getMyCounter()");
    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        23,
        new EthCallByTimestampQueryRequest(
          targetTimestamp,
          blockNumber,
          blockNumber,
          [{ to: arbitrumDemoContract, data }]
        )
      ),
    ]);
    try {
      await mock.mock(query);
    } catch (e: any) {
      expect(e.message).toMatch(
        "eth_call_by_timestamp query blocks are not adjacent"
      );
    }
  });
  test("EthCallByTimestampQueryRequest fails with wrong timestamp", async () => {
    expect.assertions(1);
    const targetTimestamp =
      BigInt(Date.now() - 1000 * 60 * 30) * // thirty minutes ago
      BigInt(1000); // milliseconds to microseconds
    const blockNumber = (
      await axios.post(ARBITRUM_NODE_URL, {
        jsonrpc: "2.0",
        id: 1,
        method: "eth_getBlockByNumber",
        params: ["finalized", false],
      })
    ).data?.result?.number;
    const arbitrumDemoContract = "0x6E36177f26A3C9cD2CE8DDF1b12904fe36deA47F";
    const data = eth.abi.encodeFunctionSignature("getMyCounter()");
    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        23,
        new EthCallByTimestampQueryRequest(
          targetTimestamp,
          blockNumber,
          `0x${(parseInt(blockNumber, 16) + 1).toString(16)}`,
          [{ to: arbitrumDemoContract, data }]
        )
      ),
    ]);
    try {
      await mock.mock(query);
    } catch (e: any) {
      expect(e.message).toMatch(
        "eth_call_by_timestamp desired timestamp falls outside of block range"
      );
    }
  });
  test("SolAccount to devnet", async () => {
    const accounts = [
      "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
      "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
    ];

    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaAccountQueryRequest("finalized", accounts)
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d0000e8890423c78a0901000000000000000000000000000000000000000000000000000000000000000000000000"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d01000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000"
    );
  });
  test("SolAccount to devnet with min context slot", async () => {
    const accounts = [
      "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
      "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
    ];

    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaAccountQueryRequest("finalized", accounts, BigInt(7))
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d0000e8890423c78a0901000000000000000000000000000000000000000000000000000000000000000000000000"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d01000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000"
    );
  });
  test("SolAccount to devnet with data slice", async () => {
    const accounts = [
      "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
      "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
    ];

    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaAccountQueryRequest(
          "finalized",
          accounts,
          BigInt(0),
          BigInt(1),
          BigInt(10)
        )
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "000000574108aed69daf"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "000000574108aed69daf"
    );
  });
  test("SolAccount to devnet with min context slot and data slice", async () => {
    const accounts = [
      "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
      "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
    ];

    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaAccountQueryRequest(
          "finalized",
          accounts,
          BigInt(7),
          BigInt(1),
          BigInt(10)
        )
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "000000574108aed69daf"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "000000574108aed69daf"
    );
  });
  test("SolanaPda to devnet", async () => {
    const query = new QueryRequest(42, [
      new PerChainQueryRequest(
        1,
        new SolanaPdaQueryRequest(
          "finalized",
          SOL_PDAS,
          BigInt(0),
          BigInt(12),
          BigInt(16) // After this, things can change.
        )
      ),
    ]);
    const resp = await mock.mock(query);
    const queryResponse = QueryResponse.from(resp.bytes);
    const sar = queryResponse.responses[0].response as SolanaPdaQueryResponse;
    expect(sar.blockTime).not.toEqual(BigInt(0));
    expect(sar.results.length).toEqual(1);

    expect(Buffer.from(sar.results[0].account).toString("hex")).toEqual(
      "4fa9188b339cfd573a0778c5deaeeee94d4bcfb12b345bf8e417e5119dae773e"
    );
    expect(sar.results[0].bump).toEqual(253);
    expect(sar.results[0].lamports).toEqual(BigInt(1141440));
    expect(sar.results[0].rentEpoch).toEqual(BigInt(0));
    expect(sar.results[0].executable).toEqual(false);
    expect(Buffer.from(sar.results[0].owner).toString("hex")).toEqual(
      "02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa"
    );
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "57cd18b7f8a4d91a2da9ab4af05d0fbe"
    );
  });
});
