import { describe, expect, it } from "@jest/globals";
import { parse, serialiseVAA, Payload, VAA } from "./vaa";

describe("TokenBridgeSetPauserAddresses", () => {
  // Real testnet governance VAAs (TokenBridge action 4) submitted during the
  // WTT pauser rollout, one per runtime. Signatures are from the testnet
  // guardian; the payloads were accepted on-chain.
  const REAL_VAAS: { [name: string]: { hex: string; chain: number } } = {
    solana: {
      chain: 1,
      hex: "01000000000100179bed5737f8629a824731f9b760cf887c059f21bf572999401e0ca605df27650e6e676b1892b7204105d1965757b8b119cf9941b39722312ab8092294acfbdc01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000006a749e2400000000000000000000000000000000000000000000546f6b656e4272696467650400012083718b7ec89617b7040685e01bdcca03214022980daae91340e0c3f840c005ef2083718b7ec89617b7040685e01bdcca03214022980daae91340e0c3f840c005ef2083718b7ec89617b7040685e01bdcca03214022980daae91340e0c3f840c005ef",
    },
    sepolia: {
      chain: 10002,
      hex: "010000000001002427b3e53e0e06e48ecaac44ecd76313e382cc7d3b6edda4e0986b35630da9210783874a5b2f9de31f967e6558e15f787bb3db37fa20a21f7856cf4856a40c2901000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000006a74ab7600000000000000000000000000000000000000000000546f6b656e427269646765042712148f26a0025dccc6cfc07a7d38756280a10e295ad7148f26a0025dccc6cfc07a7d38756280a10e295ad7148f26a0025dccc6cfc07a7d38756280a10e295ad7",
    },
    sui: {
      chain: 21,
      hex: "0100000000010006a64d534f591b1b21fcfd2ee55f699c97d70450cc62f38c9590d8aaa8f05de2566101c532ea993c0f93a8cb9a2e359ddb34a77fb0258f65762c3fd387ad5133006a75bf9d0000000000010000000000000000000000000000000000000000000000000000000000000004000000006a75bf9d00000000000000000000000000000000000000000000546f6b656e427269646765040015200c15ca93dbe1f92189ce4ce5caa5e718bdcc0e4080dd43bc255d7f30ebed64f0200c15ca93dbe1f92189ce4ce5caa5e718bdcc0e4080dd43bc255d7f30ebed64f0200c15ca93dbe1f92189ce4ce5caa5e718bdcc0e4080dd43bc255d7f30ebed64f0",
    },
  };

  it.each(Object.entries(REAL_VAAS))(
    "parses the real %s testnet VAA",
    (_name, { hex, chain }) => {
      const vaa = parse(Buffer.from(hex, "hex"));
      const payload = vaa.payload;
      if (payload.type !== "SetPauserAddresses") {
        throw new Error(`parsed as ${payload.type}`);
      }
      expect(payload.module).toBe("TokenBridge");
      expect(payload.chain).toBe(chain);
      // All three testnet deployments set the same address for every role.
      expect(payload.pauser).toBe(payload.freezer);
      expect(payload.freezer).toBe(payload.unpauser);
      // Native address size: 20 bytes on EVM, 32 elsewhere.
      const expectedLen = chain === 10002 ? 2 + 40 : 2 + 64;
      expect(payload.pauser.length).toBe(expectedLen);
    }
  );

  it.each(Object.entries(REAL_VAAS))(
    "round-trips the real %s testnet VAA byte-identically",
    (_name, { hex }) => {
      const vaa = parse(Buffer.from(hex, "hex"));
      expect(vaa.payload.type).toBe("SetPauserAddresses");
      expect(serialiseVAA(vaa as VAA<Payload>)).toBe(hex);
    }
  );

  it("round-trips unassigned (zero-length) roles", () => {
    const payload = {
      module: "TokenBridge",
      type: "SetPauserAddresses",
      chain: 2,
      pauser: "0x8f26a0025dccc6cfc07a7d38756280a10e295ad7",
      freezer: "",
      unpauser: "",
    } as const;
    const vaa: VAA<Payload> = {
      version: 1,
      guardianSetIndex: 0,
      signatures: [],
      timestamp: 1,
      nonce: 1,
      emitterChain: 1,
      emitterAddress:
        "0x0000000000000000000000000000000000000000000000000000000000000004",
      sequence: BigInt(1),
      consistencyLevel: 0,
      payload,
    };
    const hex = serialiseVAA(vaa);
    const reparsed = parse(Buffer.from(hex, "hex"));
    expect(reparsed.payload).toMatchObject({
      type: "SetPauserAddresses",
      chain: 2,
      pauser: "0x8f26a0025dccc6cfc07a7d38756280a10e295ad7",
      freezer: "",
      unpauser: "",
    });
    expect(serialiseVAA(reparsed as VAA<Payload>)).toBe(hex);
  });

  it("rejects trailing bytes (degrades to Other)", () => {
    const withTrailingByte = REAL_VAAS.sui.hex + "00";
    const vaa = parse(Buffer.from(withTrailingByte, "hex"));
    expect(vaa.payload.type).toBe("Other");
  });

  it("does not claim Core action 4 (TransferFees)", () => {
    // Same action number under a different module must not be shadowed.
    const payload = Buffer.concat([
      Buffer.from("Core".padStart(32, "\0"), "ascii"),
      Buffer.from([4]),
      Buffer.from([0, 2]),
      Buffer.alloc(64), // amount || recipient
    ]);
    const envelope = Buffer.concat([
      Buffer.from("010000000000", "hex"), // version, gsIndex, 0 sigs
      Buffer.from(
        "000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000100",
        "hex"
      ),
      payload,
    ]);
    const vaa = parse(envelope);
    expect(vaa.payload.type).toBe("TransferFees");
  });
});

describe("TokenBridgeSetPauserAddresses adversarial round-trips", () => {
  const HEADER =
    "010000000000" + // version, gsIndex, 0 sigs
    "000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000100";

  const MODULE_HEX = Buffer.from("TokenBridge")
    .toString("hex")
    .padStart(64, "0");

  function vaaWithPayload(payloadHex: string): Buffer {
    return Buffer.from(HEADER + payloadHex, "hex");
  }

  it("preserves a 32-zero-byte role (does NOT collapse to zero-length)", () => {
    // The Solana canonical serializer emits an unassigned role as 32 zero
    // bytes rather than zero-length; both are legal and must round-trip
    // byte-identically (digest stability).
    const zeros64 = "00".repeat(32);
    const payloadHex =
      MODULE_HEX +
      "04" +
      "0001" +
      "20" +
      zeros64 +
      "20" +
      zeros64 +
      "20" +
      zeros64;
    const buf = vaaWithPayload(payloadHex);
    const vaa = parse(buf);
    expect(vaa.payload.type).toBe("SetPauserAddresses");
    const p: any = vaa.payload;
    expect(p.pauser).toBe("0x" + zeros64);
    expect(serialiseVAA(vaa as VAA<Payload>)).toBe(buf.toString("hex"));
  });

  it("preserves an on-chain-invalid length (e.g. 5 bytes) byte-identically", () => {
    const payloadHex =
      MODULE_HEX + "04" + "0002" + "05" + "aabbccddee" + "00" + "00";
    const buf = vaaWithPayload(payloadHex);
    const vaa = parse(buf);
    expect(vaa.payload.type).toBe("SetPauserAddresses");
    expect(serialiseVAA(vaa as VAA<Payload>)).toBe(buf.toString("hex"));
  });

  it("degrades to Other when a role length overruns the buffer", () => {
    const payloadHex = MODULE_HEX + "04" + "0002" + "20" + "00".repeat(10);
    const vaa = parse(vaaWithPayload(payloadHex));
    expect(vaa.payload.type).toBe("Other");
  });

  it("degrades to Other when fewer than 3 roles are present", () => {
    const payloadHex = MODULE_HEX + "04" + "0002" + "00" + "00"; // only 2 roles
    const vaa = parse(vaaWithPayload(payloadHex));
    expect(vaa.payload.type).toBe("Other");
  });

  it("round-trips chain 0 and 65535", () => {
    for (const chainHex of ["0000", "ffff"]) {
      const payloadHex = MODULE_HEX + "04" + chainHex + "00" + "00" + "00";
      const buf = vaaWithPayload(payloadHex);
      const vaa = parse(buf);
      expect(vaa.payload.type).toBe("SetPauserAddresses");
      expect(serialiseVAA(vaa as VAA<Payload>)).toBe(buf.toString("hex"));
    }
  });

  it("does not let SetPauserAddresses shadow TokenBridge RegisterChain (action 1)", () => {
    // module || action=1 || chain || emitterChain || emitterAddress
    const payloadHex = MODULE_HEX + "01" + "0000" + "0001" + "11".repeat(32);
    const vaa = parse(vaaWithPayload(payloadHex));
    expect(vaa.payload.type).toBe("RegisterChain");
  });

  it("NFTBridge module with action 4 is NOT claimed as SetPauserAddresses", () => {
    const nftModuleHex = Buffer.from("NFTBridge")
      .toString("hex")
      .padStart(64, "0");
    const payloadHex = nftModuleHex + "04" + "0002" + "00" + "00" + "00";
    const vaa = parse(vaaWithPayload(payloadHex));
    expect(vaa.payload.type).toBe("Other");
  });
});
