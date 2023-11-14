import { describe, expect, test } from "@jest/globals";
import {
  ChainQueryType,
  EthCallByTimestampQueryRequest,
  EthCallByTimestampQueryResponse,
  EthCallQueryRequest,
  EthCallQueryResponse,
  EthCallWithFinalityQueryRequest,
  EthCallWithFinalityQueryResponse,
  PerChainQueryRequest,
  PerChainQueryResponse,
  QueryRequest,
  QueryResponse,
} from "..";

describe("from works with hex and Uint8Array", () => {
  test("QueryResponse", () => {
    const result =
      "010000b094a2ee9b1d5b310e1710bb5f6106bd481f28d932f83d6220c8ffdd5c55b91818ca78c812cd03c51338e384ab09265aa6fb2615a830e13c65775e769c5505800100000037010000002a010005010000002a0000000930783238343236626201130db1b83d205562461ed0720b37f1fbc21bf67f00000004916d5743010005010000005500000000028426bb7e422fe7df070cd5261d8e23280debfd1ac8c544dcd80837c5f1ebda47c06b7f000609c35ffdb8800100000020000000000000000000000000000000000000000000000000000000000000002a";
    const queryResponseFromHex = QueryResponse.from(result);
    const queryResponseFromUint8Array = QueryResponse.from(
      Buffer.from(result, "hex")
    );
    expect(queryResponseFromHex.serialize()).toEqual(
      queryResponseFromUint8Array.serialize()
    );
  });
});
describe("from yields known results", () => {
  test("demo contract call", () => {
    const result =
      "010000b094a2ee9b1d5b310e1710bb5f6106bd481f28d932f83d6220c8ffdd5c55b91818ca78c812cd03c51338e384ab09265aa6fb2615a830e13c65775e769c5505800100000037010000002a010005010000002a0000000930783238343236626201130db1b83d205562461ed0720b37f1fbc21bf67f00000004916d5743010005010000005500000000028426bb7e422fe7df070cd5261d8e23280debfd1ac8c544dcd80837c5f1ebda47c06b7f000609c35ffdb8800100000020000000000000000000000000000000000000000000000000000000000000002a";
    const queryResponse = QueryResponse.from(Buffer.from(result, "hex"));
    expect(queryResponse.version).toEqual(1);
    expect(queryResponse.requestChainId).toEqual(0);
    expect(queryResponse.requestId).toEqual(
      "0xb094a2ee9b1d5b310e1710bb5f6106bd481f28d932f83d6220c8ffdd5c55b91818ca78c812cd03c51338e384ab09265aa6fb2615a830e13c65775e769c55058001"
    );
    expect(queryResponse.request.version).toEqual(1);
    expect(queryResponse.request.nonce).toEqual(42);
    expect(queryResponse.request.requests.length).toEqual(1);
    expect(queryResponse.request.requests[0].chainId).toEqual(5);
    expect(queryResponse.request.requests[0].query.type()).toEqual(
      ChainQueryType.EthCall
    );
    expect(
      (queryResponse.request.requests[0].query as EthCallQueryRequest).blockTag
    ).toEqual("0x28426bb");
    expect(
      (queryResponse.request.requests[0].query as EthCallQueryRequest).callData
        .length
    ).toEqual(1);
    expect(
      (queryResponse.request.requests[0].query as EthCallQueryRequest)
        .callData[0].to
    ).toEqual("0x130db1b83d205562461ed0720b37f1fbc21bf67f");
    expect(
      (queryResponse.request.requests[0].query as EthCallQueryRequest)
        .callData[0].data
    ).toEqual("0x916d5743");
    expect(queryResponse.responses.length).toEqual(1);
    expect(queryResponse.responses[0].chainId).toEqual(5);
    expect(queryResponse.responses[0].response.type()).toEqual(
      ChainQueryType.EthCall
    );
    expect(
      (queryResponse.responses[0].response as EthCallQueryResponse).blockNumber
    ).toEqual(BigInt(42215099));
    expect(
      (queryResponse.responses[0].response as EthCallQueryResponse).blockHash
    ).toEqual(
      "0x7e422fe7df070cd5261d8e23280debfd1ac8c544dcd80837c5f1ebda47c06b7f"
    );
    expect(
      (queryResponse.responses[0].response as EthCallQueryResponse).blockTime
    ).toEqual(BigInt(1699584594000000));
    expect(
      (queryResponse.responses[0].response as EthCallQueryResponse).results
        .length
    ).toEqual(1);
    expect(
      (queryResponse.responses[0].response as EthCallQueryResponse).results[0]
    ).toEqual(
      "0x000000000000000000000000000000000000000000000000000000000000002a"
    );
  });
});

describe("serialize and from are inverse", () => {
  test("EthCallQueryResponse", () => {
    const serializedResponse = new EthCallQueryResponse(
      BigInt("12344321"),
      "0x123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123a",
      BigInt("789789789"),
      ["0xdeadbeef", "0x00", "0x01"]
    ).serialize();
    expect(EthCallQueryResponse.from(serializedResponse).serialize()).toEqual(
      serializedResponse
    );
  });
  test("EthCallByTimestampQueryResponse", () => {
    const serializedResponse = new EthCallByTimestampQueryResponse(
      BigInt("12344321"),
      "0x123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123a",
      BigInt("789789789"),
      BigInt("12344321"),
      "0x123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123a",
      BigInt("789789789"),
      ["0xdeadbeef", "0x00", "0x01"]
    ).serialize();
    expect(
      EthCallByTimestampQueryResponse.from(serializedResponse).serialize()
    ).toEqual(serializedResponse);
  });
  test("EthCallWithFinalityQueryResponse", () => {
    const serializedResponse = new EthCallWithFinalityQueryResponse(
      BigInt("12344321"),
      "0x123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123a",
      BigInt("789789789"),
      ["0xdeadbeef", "0x00", "0x01"]
    ).serialize();
    expect(
      EthCallWithFinalityQueryResponse.from(serializedResponse).serialize()
    ).toEqual(serializedResponse);
  });
  const exampleQueryRequest = new QueryRequest(42, [
    new PerChainQueryRequest(
      5,
      new EthCallQueryRequest(987654, [
        {
          to: "0x130Db1B83d205562461eD0720B37f1FBC21Bf67F",
          data: "0x01234567",
        },
      ])
    ),
    new PerChainQueryRequest(
      2,
      new EthCallByTimestampQueryRequest(BigInt(99999999), 12345, 45678, [
        {
          to: "0x130Db1B83d205562461eD0720B37f1FBC21Bf67F",
          data: "0x01234567",
        },
      ])
    ),
    new PerChainQueryRequest(
      23,
      new EthCallWithFinalityQueryRequest(987654, "finalized", [
        {
          to: "0x130Db1B83d205562461eD0720B37f1FBC21Bf67F",
          data: "0x01234567",
        },
      ])
    ),
  ]);
  test("QueryRequest", () => {
    const serializedRequest = exampleQueryRequest.serialize();
    expect(QueryRequest.from(serializedRequest).serialize()).toEqual(
      serializedRequest
    );
  });
  test("QueryResponse", () => {
    const serializedResponse = new QueryResponse(
      0,
      Buffer.from(new Array(65)).toString("hex"),
      exampleQueryRequest,
      [
        new PerChainQueryResponse(
          5,
          new EthCallQueryResponse(
            BigInt(987654),
            "0x123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123a",
            BigInt(99998888),
            ["0xdeadbeef", "0x00", "0x01"]
          )
        ),
        new PerChainQueryResponse(
          2,
          new EthCallByTimestampQueryResponse(
            BigInt(987654),
            "0x123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123a",
            BigInt(99998888),
            BigInt(987654),
            "0x123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123a",
            BigInt(99998888),
            ["0xdeadbeef", "0x00", "0x01"]
          )
        ),
        new PerChainQueryResponse(
          23,
          new EthCallWithFinalityQueryResponse(
            BigInt(987654),
            "0x123abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123a",
            BigInt(99998888),
            ["0xdeadbeef", "0x00", "0x01"]
          )
        ),
      ]
    ).serialize();
    expect(QueryResponse.from(serializedResponse).serialize()).toEqual(
      serializedResponse
    );
  });
});
