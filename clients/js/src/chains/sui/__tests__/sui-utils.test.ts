import { toHex } from "@mysten/sui/utils";
import { SuiGrpcClient } from "@mysten/sui/grpc";
import {
  getPublishedPackageId,
  normalizeSuiAddress,
  normalizeSuiType,
  toSuiTransactionResult,
} from "../utils";
import {
  CoinTypeKeyBcs,
  getOriginalAssetSui,
  isValidSuiAddress,
  isValidSuiType,
  trimSuiType,
} from "../../../sdk/sui";

// The gRPC envelope shape consumed by toSuiTransactionResult. Cast at the call
// site since hand-building the SDK's include-parameterized type is unnecessary
// for exercising the mapping.
const grpcTx = (tx: unknown) =>
  toSuiTransactionResult(tx as Parameters<typeof toSuiTransactionResult>[0]);

describe("toSuiTransactionResult (gRPC effects -> SuiTransactionResult)", () => {
  it("maps changedObjects, owners, created/package flags, and events", () => {
    const res = grpcTx({
      digest: "DIGEST1",
      status: { success: true, error: null },
      transaction: { sender: "0xsender" },
      objectTypes: { "0xstate": "0xpkg::state::State" },
      effects: {
        changedObjects: [
          {
            objectId: "0xpkg",
            outputOwner: { $kind: "Immutable" },
            idOperation: "Created",
            outputState: "PackageWrite",
          },
          {
            objectId: "0xstate",
            outputOwner: { $kind: "Shared" },
            idOperation: "Created",
            outputState: "ObjectWrite",
          },
          {
            objectId: "0xgas",
            outputOwner: { $kind: "AddressOwner", AddressOwner: "0xowner" },
            idOperation: "None",
            outputState: "ObjectWrite",
          },
        ],
      },
      events: [
        {
          packageId: "0xpkg",
          module: "publish_message",
          sender: "0xsender",
          eventType: "0xpkg::m::E",
          json: { a: 1 },
        },
      ],
    });

    expect(res.digest).toBe("DIGEST1");
    expect(res.success).toBe(true);
    expect(res.error).toBeUndefined();
    expect(res.sender).toBe("0xsender");
    expect(res.changedObjects).toEqual([
      {
        objectId: "0xpkg",
        type: undefined,
        owner: "Immutable",
        created: true,
        isPackage: true,
      },
      {
        objectId: "0xstate",
        type: "0xpkg::state::State",
        owner: "Shared",
        created: true,
        isPackage: false,
      },
      {
        objectId: "0xgas",
        type: undefined,
        owner: "0xowner",
        created: false,
        isPackage: false,
      },
    ]);
    expect(res.events).toEqual([
      {
        packageId: "0xpkg",
        module: "publish_message",
        sender: "0xsender",
        eventType: "0xpkg::m::E",
        json: { a: 1 },
      },
    ]);
    // The single PackageWrite is resolvable as the published package.
    expect(getPublishedPackageId(res)).toBe("0xpkg");
  });

  it("surfaces the failure status message as the error", () => {
    const res = grpcTx({
      digest: "DIGEST2",
      status: { success: false, error: { message: "MoveAbort" } },
      effects: { changedObjects: [] },
      events: [],
    });
    expect(res.success).toBe(false);
    expect(res.error).toBe("MoveAbort");
  });
});

describe("Sui type/address helpers", () => {
  it("normalizes addresses to 32-byte 0x form", () => {
    expect(normalizeSuiAddress("0x2")).toBe(`0x${"0".repeat(63)}2`);
  });

  it("normalizes fully-qualified types", () => {
    expect(normalizeSuiType("0x2::sui::SUI")).toBe(
      `0x${"0".repeat(63)}2::sui::SUI`
    );
  });

  it("validates Sui addresses and types", () => {
    expect(isValidSuiAddress("0x2")).toBe(true);
    expect(isValidSuiAddress("not-an-address")).toBe(false);
    expect(isValidSuiType("0x2::sui::SUI")).toBe(true);
    expect(isValidSuiType("0x2::sui")).toBe(false);
  });

  it("trims leading zeroes in types", () => {
    expect(trimSuiType(`0x${"0".repeat(63)}2::sui::SUI`)).toBe("0x2::sui::SUI");
  });
});

describe("CoinTypeKey BCS encoding", () => {
  // Ground truth captured from the Sui mainnet token bridge `coin_types` table:
  // chain 21 (Sui), 32-byte address, encoded as `15 00` (u16 LE) + `20` (vector
  // length) + 32 address bytes.
  it("matches the on-chain dynamic-field key bytes", () => {
    const addrHex =
      "2a62e389553ae6f061970ce1be2607c7f918154532e4296512d5a2c773424ff5";
    const addr = Array.from(Buffer.from(addrHex, "hex"));
    const encoded = toHex(
      CoinTypeKeyBcs.serialize({ chain: 21, addr }).toBytes()
    );
    expect(encoded).toBe(`150020${addrHex}`);
  });
});

describe("getOriginalAssetSui", () => {
  const mockClient = {
    getObject: jest.fn(),
    getDynamicField: jest.fn(),
  } as unknown as SuiGrpcClient;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("parses a wrapped asset from token registry", async () => {
    const tokenBridgeStateId =
      "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42";
    const registryId =
      "0x8f52cdc82ea6b81c8bfbffa76a82ee88076b9a2e89c6ba08b5bb7e9a2badc00c";
    const coinType =
      "0x1234567890123456789012345678901234567890123456789012345678901234::coin::COIN";
    const tokenAddress = Buffer.from(
      "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
      "hex"
    );

    (mockClient.getObject as jest.Mock)
      .mockResolvedValueOnce({
        object: {
          type: "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42::token_bridge::State",
          json: {
            token_registry: { id: registryId },
          },
        },
      })
      .mockResolvedValueOnce({
        object: {
          type: `0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42::wrapped_asset::WrappedAsset<${coinType}>`,
          json: {
            value: {
              info: {
                token_chain: "7",
                token_address: {
                  value: {
                    data: Buffer.from(tokenAddress).toString("base64"),
                  },
                },
              },
            },
          },
        },
      });

    (mockClient.getDynamicField as jest.Mock).mockResolvedValueOnce({
      dynamicField: {
        fieldId: "0x" + "a".repeat(64),
      },
    });

    const result = await getOriginalAssetSui(
      mockClient,
      tokenBridgeStateId,
      coinType
    );

    expect(result).toEqual({
      isWrapped: true,
      chainId: 7,
      assetAddress: new Uint8Array(tokenAddress),
    });

    expect(mockClient.getObject).toHaveBeenCalledTimes(2);
    expect(mockClient.getDynamicField).toHaveBeenCalledTimes(1);
  });

  it("parses a native asset from token registry", async () => {
    const tokenBridgeStateId =
      "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42";
    const registryId =
      "0x8f52cdc82ea6b81c8bfbffa76a82ee88076b9a2e89c6ba08b5bb7e9a2badc00c";
    const coinType = "0x2::sui::SUI";
    const tokenAddress = Buffer.alloc(32, 0);

    (mockClient.getObject as jest.Mock)
      .mockResolvedValueOnce({
        object: {
          type: "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42::token_bridge::State",
          json: {
            token_registry: { id: registryId },
          },
        },
      })
      .mockResolvedValueOnce({
        object: {
          type: `0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42::native_asset::NativeAsset<${coinType}>`,
          json: {
            value: {
              token_address: {
                value: {
                  data: Buffer.from(tokenAddress).toString("base64"),
                },
              },
            },
          },
        },
      });

    (mockClient.getDynamicField as jest.Mock).mockResolvedValueOnce({
      dynamicField: {
        fieldId: "0x" + "b".repeat(64),
      },
    });

    const result = await getOriginalAssetSui(
      mockClient,
      tokenBridgeStateId,
      coinType
    );

    expect(result).toEqual({
      isWrapped: false,
      chainId: 21,
      assetAddress: new Uint8Array(tokenAddress),
    });
  });

  it("throws on invalid coin type", async () => {
    const tokenBridgeStateId =
      "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42";

    await expect(
      getOriginalAssetSui(mockClient, tokenBridgeStateId, "invalid::type")
    ).rejects.toThrow("Invalid Sui type");

    expect(mockClient.getObject).not.toHaveBeenCalled();
  });

  it("throws when token is not registered", async () => {
    const tokenBridgeStateId =
      "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42";
    const registryId =
      "0x8f52cdc82ea6b81c8bfbffa76a82ee88076b9a2e89c6ba08b5bb7e9a2badc00c";
    const coinType =
      "0x1234567890123456789012345678901234567890123456789012345678901234::coin::COIN";

    (mockClient.getObject as jest.Mock).mockResolvedValueOnce({
      object: {
        type: "0x26efee2b51c911237888e5dc6702868abca3c7ac12c53f76ef8dc369bad5de42::token_bridge::State",
        json: {
          token_registry: { id: registryId },
        },
      },
    });

    (mockClient.getDynamicField as jest.Mock).mockRejectedValueOnce(
      new Error("not found")
    );

    await expect(
      getOriginalAssetSui(mockClient, tokenBridgeStateId, coinType)
    ).rejects.toThrow("has not been registered with the token bridge");
  });
});
