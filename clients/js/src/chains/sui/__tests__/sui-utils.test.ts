import { toHex } from "@mysten/sui/utils";
import {
  getCreatedObjects,
  getPublishedPackageId,
  normalizeSuiAddress,
  normalizeSuiType,
  SuiTransactionResult,
  toSuiTransactionResult,
} from "../utils";
import {
  CoinTypeKeyBcs,
  isValidSuiAddress,
  isValidSuiType,
  trimSuiType,
} from "../../../sdk/sui";

const baseResult = (
  changedObjects: SuiTransactionResult["changedObjects"]
): SuiTransactionResult => ({
  digest: "DiGeStXyZ",
  success: true,
  changedObjects,
  events: [],
});

describe("Sui transaction-result transforms", () => {
  it("getCreatedObjects returns created non-package objects with their type", () => {
    const res = baseResult([
      {
        objectId: "0xstate",
        type: "0xpkg::state::State",
        owner: "Shared",
        created: true,
        isPackage: false,
      },
      {
        objectId: "0xpackage",
        type: undefined,
        owner: "Immutable",
        created: true,
        isPackage: true,
      },
      {
        objectId: "0xmutated",
        type: "0xpkg::thing::Thing",
        owner: "0xowner",
        created: false,
        isPackage: false,
      },
    ]);

    expect(getCreatedObjects(res)).toEqual([
      { type: "0xpkg::state::State", objectId: "0xstate", owner: "Shared" },
    ]);
  });

  it("getPublishedPackageId returns the single published package", () => {
    const res = baseResult([
      {
        objectId: "0xpackage",
        type: undefined,
        owner: "Immutable",
        created: true,
        isPackage: true,
      },
      {
        objectId: "0xobj",
        type: "0xpkg::state::State",
        owner: "Shared",
        created: true,
        isPackage: false,
      },
    ]);

    expect(getPublishedPackageId(res)).toBe("0xpackage");
  });

  it("getPublishedPackageId throws when the package count is not exactly one", () => {
    expect(() => getPublishedPackageId(baseResult([]))).toThrow();
  });
});

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
