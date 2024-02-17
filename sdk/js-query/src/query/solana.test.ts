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
import base58 from "bs58";
import {
  ChainQueryType,
  SolanaAccountQueryRequest,
  SolanaAccountQueryResponse,
  SolanaAccountResult,
  SolanaPdaEntry,
  PerChainQueryRequest,
  QueryRequest,
  sign,
  QueryResponse,
} from "..";

jest.setTimeout(125000);

const CI = process.env.CI;
const ENV = "DEVNET";
const SERVER_URL = CI ? "http://query-server:" : "http://localhost:";
const CCQ_SERVER_URL = SERVER_URL + "6069/v1";
const QUERY_URL = CCQ_SERVER_URL + "/query";
const SOLANA_NODE_URL = CI
  ? "http://solana-devnet:8899"
  : "http://localhost:8899";

const PRIVATE_KEY =
  "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";

const ACCOUNTS = [
  "2WDq7wSs9zYrpx2kbHDA4RUTRch2CCTP6ZWaH4GNfnQQ", // Example token in devnet
  "BVxyYhm498L79r4HMQ9sxZ5bi41DmJmeWZ7SCS7Cyvna", // Example NFT in devnet
];

const PDAS: SolanaPdaEntry[] = [
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

async function getSolanaSlot(comm: string): Promise<bigint> {
  const response = await axios.post(SOLANA_NODE_URL, {
    jsonrpc: "2.0",
    id: 1,
    method: "getSlot",
    params: [{ commitment: comm, transactionDetails: "none" }],
  });

  return response.data.result;
}

describe("solana", () => {
  test("serialize and deserialize sol_account request with defaults", () => {
    const solAccountReq = new SolanaAccountQueryRequest("finalized", ACCOUNTS);
    expect(solAccountReq.minContextSlot).toEqual(BigInt(0));
    expect(solAccountReq.dataSliceOffset).toEqual(BigInt(0));
    expect(solAccountReq.dataSliceLength).toEqual(BigInt(0));
    const serialized = solAccountReq.serialize();
    expect(Buffer.from(serialized).toString("hex")).toEqual(
      "0000000966696e616c697a656400000000000000000000000000000000000000000000000002165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3019c006c48c8cbf33849cb07a3f936159cc523f9591cb1999abd45890ec5fee9b7"
    );
    const solAccountReq2 = SolanaAccountQueryRequest.from(serialized);
    expect(solAccountReq2).toEqual(solAccountReq);
  });
  test("serialize and deserialize sol_account request no defaults", () => {
    const minContextSlot = BigInt(123456);
    const dataSliceOffset = BigInt(12);
    const dataSliceLength = BigInt(100);
    const solAccountReq = new SolanaAccountQueryRequest(
      "finalized",
      ACCOUNTS,
      minContextSlot,
      dataSliceOffset,
      dataSliceLength
    );
    expect(solAccountReq.minContextSlot).toEqual(minContextSlot);
    expect(solAccountReq.dataSliceOffset).toEqual(dataSliceOffset);
    expect(solAccountReq.dataSliceLength).toEqual(dataSliceLength);
    const serialized = solAccountReq.serialize();
    expect(Buffer.from(serialized).toString("hex")).toEqual(
      "0000000966696e616c697a6564000000000001e240000000000000000c000000000000006402165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3019c006c48c8cbf33849cb07a3f936159cc523f9591cb1999abd45890ec5fee9b7"
    );
    const solAccountReq2 = SolanaAccountQueryRequest.from(serialized);
    expect(solAccountReq2).toEqual(solAccountReq);
  });
  test("serialize and deserialize sol_account response", () => {
    const slotNumber = BigInt(240866260);
    const blockTime = BigInt(1704770509);
    const blockHash = Uint8Array.from(
      Buffer.from(
        "9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3",
        "hex"
      )
    );
    const results: SolanaAccountResult[] = [
      {
        lamports: BigInt(1141440),
        rentEpoch: BigInt(123456),
        executable: false,
        owner: Uint8Array.from(
          Buffer.from(
            "02a8f6914e88a16e395ae128948ffa695693376818dd47435221f3c600000000",
            "hex"
          )
        ),
        data: Uint8Array.from(
          Buffer.from(
            "0200000062d14b7d0e121f8575cce871896548fe26d2899b0578ec92117440cda609b010",
            "hex"
          )
        ),
      },
      {
        lamports: BigInt(1141441),
        rentEpoch: BigInt(123457),
        executable: true,
        owner: Uint8Array.from(
          Buffer.from(
            "02a8f6914e88a16e395ae128948ffa695693376818dd47435221f3c600000000",
            "hex"
          )
        ),
        data: Uint8Array.from(
          Buffer.from(
            "0200000083f7752f3b75f905f040f0087c67c47a52272fcfa90e691ea6e8d4362039ecd5",
            "hex"
          )
        ),
      },
    ];
    const solAccountResp = new SolanaAccountQueryResponse(
      slotNumber,
      blockTime,
      blockHash,
      results
    );
    expect(solAccountResp.slotNumber).toEqual(slotNumber);
    expect(solAccountResp.blockTime).toEqual(blockTime);
    expect(solAccountResp.results).toEqual(results);
    const serialized = solAccountResp.serialize();
    expect(Buffer.from(serialized).toString("hex")).toEqual(
      "000000000e5b53d400000000659cbbcd9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3020000000000116ac0000000000001e2400002a8f6914e88a16e395ae128948ffa695693376818dd47435221f3c600000000000000240200000062d14b7d0e121f8575cce871896548fe26d2899b0578ec92117440cda609b0100000000000116ac1000000000001e2410102a8f6914e88a16e395ae128948ffa695693376818dd47435221f3c600000000000000240200000083f7752f3b75f905f040f0087c67c47a52272fcfa90e691ea6e8d4362039ecd5"
    );
    const solAccountResp2 = SolanaAccountQueryResponse.from(serialized);
    expect(solAccountResp2).toEqual(solAccountResp);
  });
  test("successful sol_account query", async () => {
    const solAccountReq = new SolanaAccountQueryRequest("finalized", ACCOUNTS);
    const nonce = 42;
    const query = new PerChainQueryRequest(1, solAccountReq);
    const request = new QueryRequest(nonce, [query]);
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

    const queryResponse = QueryResponse.from(response.data.bytes);
    expect(queryResponse.version).toEqual(1);
    expect(queryResponse.requestChainId).toEqual(0);
    expect(queryResponse.request.version).toEqual(1);
    expect(queryResponse.request.requests.length).toEqual(1);
    expect(queryResponse.request.requests[0].chainId).toEqual(1);
    expect(queryResponse.request.requests[0].query.type()).toEqual(
      ChainQueryType.SolanaAccount
    );

    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(sar.slotNumber).not.toEqual(BigInt(0));
    expect(sar.blockTime).not.toEqual(BigInt(0));
    expect(sar.results.length).toEqual(2);

    expect(sar.results[0].lamports).toEqual(BigInt(1461600));
    expect(sar.results[0].rentEpoch).toEqual(BigInt(0));
    expect(sar.results[0].executable).toEqual(false);
    expect(base58.encode(Buffer.from(sar.results[0].owner))).toEqual(
      "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
    );
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d0000e8890423c78a0901000000000000000000000000000000000000000000000000000000000000000000000000"
    );

    expect(sar.results[1].lamports).toEqual(BigInt(1461600));
    expect(sar.results[1].rentEpoch).toEqual(BigInt(0));
    expect(sar.results[1].executable).toEqual(false);
    expect(base58.encode(Buffer.from(sar.results[1].owner))).toEqual(
      "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d01000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000"
    );
  });
  test("sol_account query with future min context slot", async () => {
    const currSlot = await getSolanaSlot("finalized");
    const minContextSlot = BigInt(currSlot) + BigInt(10);
    const solAccountReq = new SolanaAccountQueryRequest(
      "finalized",
      ACCOUNTS,
      minContextSlot
    );
    const nonce = 42;
    const query = new PerChainQueryRequest(1, solAccountReq);
    const request = new QueryRequest(nonce, [query]);
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

    const queryResponse = QueryResponse.from(response.data.bytes);
    expect(queryResponse.version).toEqual(1);
    expect(queryResponse.requestChainId).toEqual(0);
    expect(queryResponse.request.version).toEqual(1);
    expect(queryResponse.request.requests.length).toEqual(1);
    expect(queryResponse.request.requests[0].chainId).toEqual(1);
    expect(queryResponse.request.requests[0].query.type()).toEqual(
      ChainQueryType.SolanaAccount
    );

    const sar = queryResponse.responses[0]
      .response as SolanaAccountQueryResponse;
    expect(sar.slotNumber).toEqual(minContextSlot);
    expect(sar.blockTime).not.toEqual(BigInt(0));
    expect(sar.results.length).toEqual(2);

    expect(sar.results[0].lamports).toEqual(BigInt(1461600));
    expect(sar.results[0].rentEpoch).toEqual(BigInt(0));
    expect(sar.results[0].executable).toEqual(false);
    expect(base58.encode(Buffer.from(sar.results[0].owner))).toEqual(
      "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
    );
    expect(Buffer.from(sar.results[0].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d0000e8890423c78a0901000000000000000000000000000000000000000000000000000000000000000000000000"
    );

    expect(sar.results[1].lamports).toEqual(BigInt(1461600));
    expect(sar.results[1].rentEpoch).toEqual(BigInt(0));
    expect(sar.results[1].executable).toEqual(false);
    expect(base58.encode(Buffer.from(sar.results[1].owner))).toEqual(
      "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
    );
    expect(Buffer.from(sar.results[1].data).toString("hex")).toEqual(
      "01000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d01000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000"
    );
  });
});
