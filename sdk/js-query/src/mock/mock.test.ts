import {
  afterAll,
  beforeAll,
  describe,
  expect,
  jest,
  test,
} from "@jest/globals";
import axios from "axios";
import { eth } from "web3";
import {
  EthCallQueryRequest,
  EthCallWithFinalityQueryRequest,
  PerChainQueryRequest,
  QueryProxyMock,
  QueryProxyQueryResponse,
  QueryRequest,
} from "..";

jest.setTimeout(60000);

const POLYGON_NODE_URL = "https://polygon-mumbai-bor.publicnode.com";
const ARBITRUM_NODE_URL = "https://arbitrum-goerli.publicnode.com";
const QUERY_URL = "https://testnet.ccq.vaa.dev/v1/query";

let mock: QueryProxyMock;

beforeAll(() => {
  mock = new QueryProxyMock({
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
});
