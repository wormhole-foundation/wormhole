import { spy } from "@certusone/wormhole-sdk-proto-node";
import * as grpc from "@grpc/grpc-js";
import { fc } from "@fast-check/vitest";
import { keccak_256 } from "@noble/hashes/sha3";
import * as secp from "@noble/secp256k1";
import { Cl, ClarityValue } from "@stacks/transactions";
import { toBigIntBE, toBufferBE } from "bigint-buffer";
import { webcrypto } from "node:crypto";

// @ts-ignore
if (!globalThis.crypto) globalThis.crypto = webcrypto;

import { hmac } from "@noble/hashes/hmac";
import { sha256 } from "@noble/hashes/sha256";
import { hexToBytes } from "@noble/hashes/utils";
import { createClient } from "@stacks/blockchain-api-client";
import { promisify } from "util";
import { CHAIN_ID_STACKS, SPY_SERVICE_HOST, STACKS_API_URL } from "./constants";

const sleep = promisify(setTimeout);

export type ReadinessCheck = () => Promise<boolean>;

// STACKS

export async function waitForTransactionSuccess(
  txid: string,
  timeoutMs = 30_000
): Promise<void> {
  const api = createClient({ baseUrl: STACKS_API_URL });
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    try {
      const tx = (await api.GET("/v3/transaction/{tx_id}" as any, {
        params: { path: { tx_id: txid } },
      })) as unknown as {
        data?: { index_block_hash: string; tx: string; result: string };
        response: Response;
      };

      if (tx.data?.result.startsWith("(ok ")) {
        console.log(`Transaction ${txid}: ${tx.data.result}`);
        return;
      }

      if (tx.data?.result.startsWith("(err ")) {
        throw new Error(`Transaction ${txid} failed: ${tx.data.result}`);
      }
    } catch (error) {
      // Re-throw errors that we explicitly threw
      if (error instanceof Error && error.message.includes("failed")) {
        throw error;
      }

      console.warn(`Error waiting for tx success`, error?.message);
    }

    await sleep(1000);
  }

  throw new Error(
    `Timeout waiting for transaction ${txid} to succeed after ${timeoutMs}ms`
  );
}

// WORMHOLE

const initialGovernanceDataSource = {
  chain: 1,
  address: hexToBytes(
    "5635979a221c34931e32620b9293a463065555ea71fe97cd6237ade875b12e9e"
  ),
};

secp.etc.hmacSha256Sync = (k, ...m) =>
  hmac(sha256, k, secp.etc.concatBytes(...m));

export namespace wormhole {
  export interface Guardian {
    guardianId: number;
    secretKey: Uint8Array;
    compressedPublicKey: Uint8Array;
    uncompressedPublicKey: Uint8Array;
    ethereumAddress: Uint8Array;
  }

  export const privateKeyToGuardian = (
    privateKey: Uint8Array,
    guardianId: number = 0
  ): Guardian => {
    const uncompressedPublicKey = secp
      .getPublicKey(privateKey, false)
      .slice(1, 65);
    return {
      guardianId,
      secretKey: privateKey,
      compressedPublicKey: secp.getPublicKey(privateKey, true),
      uncompressedPublicKey,
      ethereumAddress: keccak_256(uncompressedPublicKey).slice(12, 32),
    };
  };

  export const generateGuardianSetKeychain = (count = 19): Guardian[] => {
    let keychain: Guardian[] = [];
    for (let i = 0; i < count; i++) {
      let secretKey = secp.utils.randomPrivateKey();
      let uncompressedPublicKey = secp
        .getPublicKey(secretKey, false)
        .slice(1, 65);
      let ethereumAddress = keccak_256(uncompressedPublicKey).slice(12, 32);
      keychain.push({
        guardianId: i,
        secretKey,
        uncompressedPublicKey,
        ethereumAddress,
        compressedPublicKey: secp.getPublicKey(secretKey, true),
      });
    }
    return keychain;
  };

  export interface NewEmitter {
    chain: number;
    sequence: bigint;
    address: Uint8Array;
  }

  export interface Emitter {
    chain: number;
    address: Uint8Array;
  }

  export interface VaaHeader {
    version: number;
    guardianSetId: number;
    signatures: Uint8Array[];
  }

  export interface VaaBody {
    timestamp: number;
    emitterChain: number;
    nonce: number;
    emitterAddress: Uint8Array;
    sequence: bigint;
    consistencyLevel: number;
    payload: Uint8Array;
  }

  export interface VaaHeaderBuildOptions {
    version?: number;
    guardianSetId?: number;
    signatures?: Uint8Array[];
  }

  export interface VaaBodyBuildOptions {
    timestamp?: number;
    emitterChain?: number;
    nonce?: number;
    emitterAddress?: Uint8Array;
    sequence?: bigint;
    consistencyLevel?: number;
    payload?: Uint8Array;
  }

  export const GovernanceUpdateEmitter = {
    chain: 1,
    sequence: 0,
    address: hexToBytes(
      "0000000000000000000000000000000000000000000000000000000000000004"
    ),
  };

  export namespace fc_ext {
    // Helper for generating a VAA Body;
    // Wire format reminder:
    // ===========================
    // VAA Body
    // u32         timestamp           (Timestamp of the block where the source transaction occurred)
    // u32         nonce               (A grouping number)
    // u16         emitter_chain       (Wormhole ChainId of emitter contract)
    // [32]byte    emitter_address     (Emitter contract address, in Wormhole format)
    // u64         sequence            (Strictly increasing sequence, tied to emitter address & chain)
    // u8          consistency_level   (What finality level was reached before emitting this message)
    // []byte      payload             (VAA message content)
    export const vaaBody = (opts?: VaaBodyBuildOptions) => {
      // Timestamp
      const timestamp =
        opts && opts.timestamp
          ? fc.constant(opts.timestamp)
          : fc.nat(4294967295);

      // Nonce
      const nonce =
        opts && opts.nonce ? fc.constant(opts.nonce) : fc.nat(4294967295);

      // Emitter chain
      const emitterChain =
        opts && opts.emitterChain
          ? fc.constant(opts.emitterChain)
          : fc.nat(65535);

      // Emitter address
      const emitterAddress =
        opts && opts.emitterAddress
          ? fc.constant(opts.emitterAddress)
          : fc.uint8Array({ minLength: 32, maxLength: 32 });

      // Sequence
      const sequence =
        opts && opts.sequence ? fc.constant(opts.sequence) : fc.bigInt();

      // Consistency level
      const consistencyLevel =
        opts && opts.consistencyLevel
          ? fc.constant(opts.consistencyLevel)
          : fc.nat(255);

      // Payload
      const payload =
        opts && opts.payload
          ? fc.constant(opts.payload)
          : fc.uint8Array({ minLength: 20, maxLength: 2048 });

      return fc.tuple(
        timestamp,
        nonce,
        emitterChain,
        emitterAddress,
        sequence,
        consistencyLevel,
        payload
      );
    };

    // Helper for generating a VAA Header;
    // Wire format reminder:
    // ===========================
    // VAA Header
    // byte        version             (VAA Version)
    // u32         guardian_set_index  (Indicates which guardian set is signing)
    // u8          len_signatures      (Number of signatures stored)
    // [][66]byte  signatures          (Collection of ecdsa signatures)
    export const vaaHeader = (
      opts?: VaaHeaderBuildOptions,
      numberOfSignatures = 19
    ) => {
      // Version
      const version =
        opts && opts.version ? fc.constant(opts.version) : fc.nat(255);

      // Guardian set id
      const guardianSetId =
        opts && opts.guardianSetId
          ? fc.constant(opts.guardianSetId)
          : fc.nat(255);

      // Specified signatures
      const specifiedSignatures =
        opts && opts.signatures
          ? fc.constant(opts.signatures)
          : fc.array(fc.uint8Array({ minLength: 66, maxLength: 66 }), {
              minLength: 0,
              maxLength: 0,
            });

      const specifiedSignaturesLen =
        opts && opts.signatures ? opts.signatures.length : 0;

      // Generated signatures
      let generatedSignatures = fc.array(
        fc.uint8Array({ minLength: 66, maxLength: 66 }),
        {
          minLength: 0,
          maxLength: numberOfSignatures - specifiedSignaturesLen,
        }
      );

      return fc.tuple(
        version,
        guardianSetId,
        specifiedSignatures,
        generatedSignatures
      );
    };
  }

  export const buildValidVaaHeaderSpecs = (
    keychain: Guardian[],
    body: VaaBody,
    opts?: VaaHeaderBuildOptions
  ): VaaHeaderBuildOptions => {
    let signatures: Buffer[] = [];
    const messageHash = hashVaaBody(body);

    for (let guardian of keychain) {
      const signature = secp.sign(messageHash, guardian.secretKey);

      const id = Buffer.alloc(1);
      id.writeUint8(guardian.guardianId, 0);

      // v.writeUint8(signature.addRecoveryBit, 1);
      if (signature.recovery) {
        const rec = Buffer.alloc(1);
        rec.writeUint8(signature.recovery, 0);
        signatures.push(
          Buffer.concat([
            Uint8Array.from(id),
            signature.toCompactRawBytes(),
            Uint8Array.from(rec),
          ])
        );
      } else {
        const rec = Buffer.alloc(1);
        rec.writeUint8(0, 0);
        signatures.push(
          Buffer.concat([
            Uint8Array.from(id),
            signature.toCompactRawBytes(),
            Uint8Array.from(rec),
          ])
        );
      }
    }
    return {
      version: opts?.version,
      guardianSetId: opts?.guardianSetId,
      signatures: signatures.map((x) => Uint8Array.from(x)),
    };
  };

  export const buildValidVaaHeader = (
    keychain: Guardian[],
    body: VaaBody,
    opts: VaaHeaderBuildOptions
  ): VaaHeader => {
    let specs = buildValidVaaHeaderSpecs(keychain, body, opts);
    return {
      version: specs.version!,
      guardianSetId: specs.guardianSetId!,
      signatures: specs.signatures!,
    };
  };

  export const serializeVaaToClarityValue = (
    header: VaaHeader,
    body: VaaBody,
    keychain: Guardian[]
  ): [ClarityValue, any[]] => {
    let guardiansPublicKeys: ClarityValue[] = [];
    let guardiansSignatures: ClarityValue[] = [];
    for (let i = 0; i < header.signatures.length; i++) {
      let guardianId = header.signatures[i]?.[0];
      if (keychain.length > i && guardianId !== undefined) {
        let guardian = keychain[i];
        if (guardian) {
          guardiansPublicKeys.push(
            Cl.tuple({
              "guardian-id": Cl.uint(guardianId),
              "recovered-compressed-public-key": Cl.buffer(
                guardian.compressedPublicKey
              ),
            })
          );
        }
      }
      guardiansSignatures.push(
        Cl.tuple({
          "guardian-id": Cl.uint(header.signatures[i]?.slice(0, 1) || 0),
          signature: Cl.buffer(
            header.signatures[i]?.slice(1, 66) || Buffer.alloc(65)
          ),
        })
      );
    }
    let value = Cl.tuple({
      "consistency-level": Cl.uint(body.consistencyLevel),
      version: Cl.uint(header.version),
      "guardian-set-id": Cl.uint(header.guardianSetId),
      "signatures-len": Cl.uint(header.signatures.length),
      signatures: Cl.list(guardiansSignatures),
      "emitter-chain": Cl.uint(body.emitterChain),
      "emitter-address": Cl.buffer(body.emitterAddress),
      sequence: Cl.uint(body.sequence),
      timestamp: Cl.uint(body.timestamp),
      nonce: Cl.uint(body.nonce),
      payload: Cl.buffer(body.payload),
    });
    return [value, guardiansPublicKeys];
  };

  export interface VaaBodySpec {
    values: VaaBody;
    specs: (
      | fc.Arbitrary<number>
      | fc.Arbitrary<Uint8Array>
      | fc.Arbitrary<Uint8Array[]>
      | fc.Arbitrary<bigint>
    )[];
  }

  export const buildValidVaaBodySpecs = (opts?: {
    payload?: Uint8Array;
    emitter?: Emitter;
    sequence?: bigint;
  }): VaaBody => {
    const date = Math.floor(Date.now() / 1000);
    const timestamp = date >>> 0;
    const payload =
      (opts && opts.payload && opts.payload) || new Uint8Array(32);
    let emitter =
      opts && opts.emitter ? opts.emitter : initialGovernanceDataSource;
    let values = {
      timestamp: timestamp,
      nonce: 0,
      emitterChain: emitter.chain,
      emitterAddress: emitter.address,
      sequence: opts && opts.sequence ? opts.sequence : 1n,
      consistencyLevel: 0,
      payload: payload,
    };
    return values;
  };

  export const assembleVaaBody = (
    timestamp: number | bigint | Uint8Array,
    nonce: number | bigint | Uint8Array,
    emitterChain: number | bigint | Uint8Array,
    emitterAddress: number | bigint | Uint8Array,
    sequence: number | bigint | Uint8Array,
    consistencyLevel: number | bigint | Uint8Array,
    payload: number | bigint | Uint8Array
  ): VaaBody => {
    return {
      timestamp: timestamp as number,
      nonce: nonce as number,
      emitterChain: emitterChain as number,
      emitterAddress: emitterAddress as Uint8Array,
      sequence: sequence as bigint,
      consistencyLevel: consistencyLevel as number,
      payload: payload as Uint8Array,
    };
  };

  export const assembleVaaHeader = (
    version: number | bigint | Uint8Array,
    guardianSetId: number | bigint | Uint8Array,
    signatures: number | bigint | Uint8Array[]
  ): VaaHeader => {
    return {
      version: version as number,
      guardianSetId: guardianSetId as number,
      signatures: signatures as Uint8Array[],
    };
  };

  export const hashVaaBody = (body: VaaBody): Uint8Array => {
    return keccak_256(
      keccak_256(Uint8Array.from(serializeVaaBodyToBuffer(body)))
    );
  };

  export const serializeVaaToBuffer = (
    vaaHeader: VaaHeader,
    vaaBody: VaaBody
  ) => {
    return Buffer.concat(
      [
        serializeVaaHeaderToBuffer(vaaHeader),
        serializeVaaBodyToBuffer(vaaBody),
      ].map((x) => Uint8Array.from(x))
    );
  };

  export const serializeVaaHeaderToBuffer = (vaaHeader: VaaHeader) => {
    const components: Buffer[] = [];
    var v = Buffer.alloc(1);
    v.writeUint8(vaaHeader.version, 0);
    components.push(v);

    v = Buffer.alloc(4);
    v.writeUInt32BE(vaaHeader.guardianSetId, 0);
    components.push(v);

    v = Buffer.alloc(1);
    v.writeUint8(vaaHeader.signatures.length, 0);
    components.push(v);

    components.push(Buffer.concat(vaaHeader.signatures));
    return Buffer.concat(components.map((x) => Uint8Array.from(x)));
  };

  export const serializeVaaBodyToBuffer = (vaaBody: VaaBody) => {
    const components: (Buffer | Uint8Array)[] = [];
    let v = Buffer.alloc(4);
    v.writeUInt32BE(vaaBody.timestamp, 0);
    components.push(v);

    v = Buffer.alloc(4);
    v.writeUInt32BE(vaaBody.nonce, 0);
    components.push(v);

    v = Buffer.alloc(2);
    v.writeUInt16BE(vaaBody.emitterChain, 0);
    components.push(v);

    components.push(vaaBody.emitterAddress);

    components.push(bigintToBuffer(vaaBody.sequence, 8));

    v = Buffer.alloc(1);
    v.writeUint8(vaaBody.consistencyLevel, 0);
    components.push(v);

    components.push(vaaBody.payload);

    return Buffer.concat(components.map((x) => Uint8Array.from(x)));
  };

  // This hex string represents the ASCII string "Core"
  export const coreModule = Buffer.from(
    "00000000000000000000000000000000000000000000000000000000436f7265",
    "hex"
  );

  export const validContractUpgradeModule = coreModule;
  export const validGuardianRotationModule = coreModule;
  export const validSetMessageFeeModule = coreModule;
  export const validTransferFeesModule = coreModule;

  export const serializeGuardianUpdateVaaPayloadToBuffer = (
    keyChain: Guardian[],
    action: number,
    chain: number,
    setId: number,
    module = validGuardianRotationModule
  ) => {
    const components: (Buffer | Uint8Array)[] = [];
    components.push(module);

    let v = Buffer.alloc(1);
    v.writeUint8(action, 0);
    components.push(v);

    v = Buffer.alloc(2);
    v.writeUInt16BE(chain, 0);
    components.push(v);

    v = Buffer.alloc(4);
    v.writeUInt32BE(setId, 0);
    components.push(v);

    v = Buffer.alloc(1);
    v.writeUint8(keyChain.length, 0);
    components.push(v);

    for (let guardian of keyChain) {
      components.push(guardian.ethereumAddress);
    }

    return Buffer.concat(components.map((x) => Uint8Array.from(x)));
  };

  // Serialize payload for a ContractUpgrade VAA
  //
  // Wire format reminder:
  // ===========================
  // [32]byte    module     (Module, should be "Core")
  // u8          action     (Action type, `4` for TransferFees)
  // u16         chain      (Blockchain ID which this message is intended for)
  // principal   contract   (Successor contract)
  //
  // See also: https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0004_message_publishing.md
  export const serializeContractUpgradeVaaPayloadToBuffer = (
    action: number,
    chain: number,
    contract: string,
    module = validSetMessageFeeModule
  ) => {
    const components: (Buffer | Uint8Array)[] = [];
    components.push(module);

    let v = Buffer.alloc(1);
    v.writeUint8(action, 0);
    components.push(v);

    v = Buffer.alloc(2);
    v.writeUInt16BE(chain, 0);
    components.push(v);

    // Encode `contract` as Stacks wire format
    v = Buffer.from(Cl.serialize(Cl.principal(contract)));
    components.push(v);

    return Buffer.concat(components.map((x) => Uint8Array.from(x)));
  };

  // Serialize payload for a SetMessageFee VAA
  //
  // Wire format reminder:
  // ===========================
  // [32]byte    module     (Module, should be "Core")
  // u8          action     (Action type, `3` for SetMessageFee)
  // u16         chain      (Blockchain ID which this message is intended for)
  // u256        fee        (New fee (in uSTX) to emit a message to Wormhole Guardians)
  //
  // See also: https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0004_message_publishing.md
  export const serializeSetMessageFeeVaaPayloadToBuffer = (
    action: number,
    chain: number,
    fee: number | bigint,
    module = validSetMessageFeeModule
  ) => {
    const components: Buffer[] = [];
    components.push(module);

    const v1 = Buffer.alloc(1);
    v1.writeUint8(action, 0);
    components.push(v1);

    const v2 = Buffer.alloc(2);
    v2.writeUInt16BE(chain, 0);
    components.push(v2);

    components.push(bigintToBuffer(fee, 32));

    return Buffer.concat(components.map((x) => Uint8Array.from(x)));
  };

  // Serialize payload for a TransferFees VAA
  //
  // Wire format reminder:
  // ===========================
  // [32]byte    module     (Module, should be "Core")
  // u8          action     (Action type, `4` for TransferFees)
  // u16         chain      (Blockchain ID which this message is intended for)
  // u256        amount     (Amount to transfer from accumulated message fees)
  // principal   recipient  (Recipient of message fees)
  //
  // See also: https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0004_message_publishing.md
  export const serializeTransferFeesVaaPayloadToBuffer = (
    action: number,
    chain: number,
    amount: number | bigint,
    recipient: string,
    module = validSetMessageFeeModule
  ) => {
    const components: Buffer[] = [];
    components.push(module);

    const v1 = Buffer.alloc(1);
    v1.writeUint8(action, 0);
    components.push(v1);

    const v2 = Buffer.alloc(2);
    v2.writeUInt16BE(chain, 0);
    components.push(v2);

    components.push(bigintToBuffer(amount, 32));

    // Encode `recipient` as Stacks wire format
    components.push(Buffer.from(Cl.serialize(Cl.principal(recipient))));

    return Buffer.concat(components.map((x) => Uint8Array.from(x)));
  };

  export function generateGuardianSetUpdateVaa(
    keychain: wormhole.Guardian[],
    guardianSetId: number
  ) {
    let guardianRotationPayload =
      wormhole.serializeGuardianUpdateVaaPayloadToBuffer(
        keychain,
        2,
        0,
        guardianSetId,
        wormhole.validGuardianRotationModule
      );
    let vaaBody = wormhole.buildValidVaaBodySpecs({
      payload: Uint8Array.from(guardianRotationPayload),
      emitter: wormhole.GovernanceUpdateEmitter,
    });
    let vaaHeader = wormhole.buildValidVaaHeader(keychain, vaaBody, {
      version: 1,
      guardianSetId: guardianSetId - 1,
    });
    let vaa = wormhole.serializeVaaToBuffer(vaaHeader, vaaBody);
    let uncompressedPublicKey: ClarityValue[] = [];
    for (let guardian of keychain) {
      uncompressedPublicKey.push(Cl.buffer(guardian.uncompressedPublicKey));
    }
    return {
      vaa: Uint8Array.from(vaa),
      uncompressedPublicKeys: uncompressedPublicKey,
      header: vaaHeader,
      body: vaaBody,
    };
  }

  export function generateMessageFeeVaa(
    keychain: wormhole.Guardian[],
    guardianSetId: number,
    fee: number | bigint,
    chain: number
  ) {
    const payload = wormhole.serializeSetMessageFeeVaaPayloadToBuffer(
      3,
      chain,
      fee
    );
    const body = wormhole.buildValidVaaBodySpecs({
      payload: Uint8Array.from(payload),
      emitter: wormhole.GovernanceUpdateEmitter,
    });
    const header = wormhole.buildValidVaaHeader(keychain, body, {
      version: 1,
      guardianSetId,
    });
    const vaa = wormhole.serializeVaaToBuffer(header, body);

    return {
      vaa: Uint8Array.from(vaa),
      header,
      body,
    };
  }

  // This links `wormhole-core`
}

// SPY

export interface VAAData {
  version: number;
  signatureCount: number;
  bodyOffset: number;
  emitterChain: number;
  emitterAddress: string;
  sequence: bigint;
  payload: Buffer;
}

export const parseVAABuffer = (buf: Buffer): VAAData => {
  const version = buf.readUInt8(0);
  const signatureCount = buf.readUInt8(5);
  const bodyOffset = 6 + signatureCount * 66;
  const emitterChain = buf.readUInt16BE(bodyOffset + 8);
  const emitterAddress = buf
    .subarray(bodyOffset + 10, bodyOffset + 42)
    .toString("hex");
  const sequence = buf.readBigUInt64BE(bodyOffset + 42);
  const payload = buf.subarray(bodyOffset + 51);

  return {
    version,
    signatureCount,
    bodyOffset,
    emitterChain,
    emitterAddress,
    sequence,
    payload,
  };
};

export const expectVAA = (
  expectedPayload: Buffer,
  timeoutMs = 120_000
): Promise<void> => {
  const { SpyRPCServiceClient, SubscribeSignedVAARequest } = spy;
  const client = new SpyRPCServiceClient(
    SPY_SERVICE_HOST,
    grpc.credentials.createInsecure()
  );
  const stream = client.subscribeSignedVAA(
    SubscribeSignedVAARequest.fromPartial({ filters: [] })
  );

  return new Promise<void>((resolve, reject) => {
    let vaaReceived = false;

    const cleanup = () => {
      stream.destroy();
      client.close();
    };

    const timeout = setTimeout(() => {
      if (!vaaReceived) {
        cleanup();
        reject("VAA monitoring timed out");
      }
    }, timeoutMs);

    stream.on("data", (vaa) => {
      if (vaaReceived) return;

      if (vaa.vaaBytes && vaa.vaaBytes.length > 0) {
        const vaaBuffer = Buffer.from(vaa.vaaBytes);

        if (vaaBuffer.length >= 51) {
          const signatureCount = vaaBuffer.readUInt8(5);
          const bodyOffset = 6 + signatureCount * 66;

          if (vaaBuffer.length >= bodyOffset + 51) {
            const parsedVAA = parseVAABuffer(vaaBuffer);

            // Only process VAAs from Stacks chain
            if (parsedVAA.emitterChain !== CHAIN_ID_STACKS) return;

            // Check if this VAA contains our expected payload
            if (!parsedVAA.payload.equals(expectedPayload)) return;

            console.log(`VAA received, emitter: ${parsedVAA.emitterAddress}`);

            vaaReceived = true;
            clearTimeout(timeout);
            cleanup();
            resolve();
          }
        }
      }
    });

    stream.on("error", (error: Error) => {
      clearTimeout(timeout);
      cleanup();
      reject(`VAA monitoring failed: ${error.message}`);
    });

    stream.on("end", () => {
      clearTimeout(timeout);
      cleanup();
      if (!vaaReceived) resolve();
    });
  });
};

export const expectNoStacksVAA = (timeoutMs = 60_000): Promise<void> => {
  const { SpyRPCServiceClient, SubscribeSignedVAARequest } = spy;
  const client = new SpyRPCServiceClient(
    SPY_SERVICE_HOST,
    grpc.credentials.createInsecure()
  );
  const stream = client.subscribeSignedVAA(
    SubscribeSignedVAARequest.fromPartial({ filters: [] })
  );

  return new Promise<void>((resolve, reject) => {
    let vaaReceived = false;

    const cleanup = () => {
      stream.destroy();
      client.close();
    };

    const timeout = setTimeout(() => {
      if (!vaaReceived) {
        cleanup();
        console.log("No VAA received as expected (timeout reached)");
        resolve();
      }
    }, timeoutMs);

    stream.on("data", (vaa) => {
      if (vaaReceived) return;

      if (vaa.vaaBytes && vaa.vaaBytes.length > 0) {
        const vaaBuffer = Buffer.from(vaa.vaaBytes);

        if (vaaBuffer.length >= 51) {
          const signatureCount = vaaBuffer.readUInt8(5);
          const bodyOffset = 6 + signatureCount * 66;

          if (vaaBuffer.length >= bodyOffset + 51) {
            const emitterChain = vaaBuffer.readUInt16BE(bodyOffset + 8);

            if (emitterChain === CHAIN_ID_STACKS) {
              vaaReceived = true;
              clearTimeout(timeout);
              cleanup();
              reject(
                new Error("Unexpected VAA received for faulty transaction")
              );
              return;
            }
          }
        }
      }
    });

    stream.on("error", (error: Error) => {
      clearTimeout(timeout);
      cleanup();
      reject(`VAA monitoring failed: ${error.message}`);
    });

    stream.on("end", () => {
      clearTimeout(timeout);
      cleanup();
      if (!vaaReceived) resolve();
    });
  });
};

export function bigintToBuffer(
  value: bigint | number | string,
  byteLength: number
): Buffer {
  switch (typeof value) {
    case "number":
    case "string":
      return toBufferBE(BigInt(value), byteLength);
    case "bigint":
      return toBufferBE(value, byteLength);
  }
}

export function bufferToHexString(bytes: Uint8Array) {
  return Array.from(bytes, function (byte) {
    return ("0" + (byte & 0xff).toString(16)).slice(-2);
  }).join("");
}

export function bufferToBigint(bytes: Buffer): BigInt {
  return toBigIntBE(bytes);
}
