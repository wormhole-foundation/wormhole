import { BN } from "@project-serum/anchor";
import { PublicKey } from "@solana/web3.js";
import keccak256 from "keccak256";

const elliptic = require("elliptic");

const SIGNATURE_PAYLOAD_LEN = 66;
const KEY_LENGTH = 20;

interface Guardian {
  index: number;
  key: string;
}

export class MockGuardians {
  setIndex: number;
  signers: Guardian[];

  constructor(setIndex: number, keys: string[]) {
    this.setIndex = setIndex;
    this.signers = keys.map((key, index): Guardian => {
      return { index, key };
    });
  }

  getPublicKeys() {
    return this.signers.map((guardian) => ethPrivateToPublic(guardian.key));
  }

  updateGuardianSetIndex(setIndex: number) {
    this.setIndex = setIndex;
  }

  addSignatures(message: Buffer, guardianIndices: number[]) {
    if (guardianIndices.length == 0) {
      throw Error("guardianIndices.length == 0");
    }
    const signers = this.signers.filter((signer) =>
      guardianIndices.includes(signer.index)
    );

    const sigStart = 6;
    const numSigners = signers.length;

    const signedVaa = Buffer.alloc(
      sigStart + SIGNATURE_PAYLOAD_LEN * numSigners + message.length
    );
    signedVaa.write(
      message.toString("hex"),
      sigStart + SIGNATURE_PAYLOAD_LEN * numSigners,
      "hex"
    );

    signedVaa.writeUInt8(1, 0);
    signedVaa.writeUInt32BE(this.setIndex, 1);
    signedVaa.writeUInt8(numSigners, 5);

    // signatures
    const hash = keccak256(keccak256(message));

    const ecdsa = new elliptic.ec("secp256k1");
    for (let i = 0; i < numSigners; ++i) {
      const signer = signers.at(i);
      if (signer == undefined) {
        throw Error("signer == undefined");
      }
      const key = ecdsa.keyFromPrivate(signer.key);
      const signature = key.sign(hash, { canonical: true });

      const start = sigStart + i * SIGNATURE_PAYLOAD_LEN;
      signedVaa.writeUInt8(signer.index, start);
      signedVaa.write(
        signature.r.toString(16).padStart(64, "0"),
        start + 1,
        "hex"
      );
      signedVaa.write(
        signature.s.toString(16).padStart(64, "0"),
        start + 33,
        "hex"
      );
      signedVaa.writeUInt8(signature.recoveryParam, start + 65);
    }

    return signedVaa;
  }
}

export class MockEmitter {
  chain: number;
  address: Buffer;

  sequence: number;

  constructor(emitterAddress: string, chain: number) {
    this.chain = chain;
    const address = Buffer.from(emitterAddress, "hex");
    if (address.length != 32) {
      throw Error("emitterAddress.length != 32");
    }
    this.address = address;

    this.sequence = 0;
  }

  publishMessage(
    nonce: number,
    payload: Buffer,
    consistencyLevel: number,
    timestamp?: number
  ) {
    ++this.sequence;

    const message = Buffer.alloc(51 + payload.length);

    message.writeUInt32BE(timestamp == undefined ? 0 : timestamp, 0);
    message.writeUInt32BE(nonce, 4);
    message.writeUInt16BE(this.chain, 8);
    message.write(this.address.toString("hex"), 10, "hex");
    message.writeBigUInt64BE(BigInt(this.sequence), 42);
    message.writeUInt8(consistencyLevel, 50);
    message.write(payload.toString("hex"), 51, "hex");

    return message;
  }
}

export class MockEthereumEmitter extends MockEmitter {
  constructor(emitterAddress: string, chain?: number) {
    super(emitterAddress, chain == undefined ? 2 : chain);
  }
}

export class WormholeGovernanceEmitter extends MockEmitter {
  module: Buffer;

  constructor(emitterAddress: PublicKey) {
    super(emitterAddress.toBuffer().toString("hex"), 1);

    this.module = Buffer.alloc(32);
    this.module.write("Core", 28);
  }

  publishSetMessageFee(chain: number, amount: bigint) {
    const serialized = Buffer.alloc(67);
    serialized.write(this.module.toString("hex"), 0, "hex");
    serialized.writeUInt8(3, 32); // action
    serialized.writeUInt16BE(chain, 33);

    const amountBytes = new BN(amount.toString()).toBuffer();
    serialized.write(
      amountBytes.toString("hex"),
      67 - amountBytes.length,
      "hex"
    );
    return this.publishMessage(0, serialized, 1, now());
  }

  publishTransferFees(chain: number, amount: bigint, recipient: PublicKey) {
    const serialized = Buffer.alloc(99);
    serialized.write(this.module.toString("hex"), 0, "hex");
    serialized.writeUInt8(4, 32); // action
    serialized.writeUInt16BE(chain, 33);

    const amountBytes = new BN(amount.toString()).toBuffer();
    serialized.write(
      amountBytes.toString("hex"),
      67 - amountBytes.length,
      "hex"
    );
    serialized.write(recipient.toBuffer().toString("hex"), 67, "hex");
    return this.publishMessage(0, serialized, 1, now());
  }

  publishGuardianSetUpgrade(newGuardianSetIndex: number, publicKeys: Buffer[]) {
    const numKeys = publicKeys.length;
    const serialized = Buffer.alloc(40 + KEY_LENGTH * numKeys);
    serialized.write(this.module.toString("hex"), 0, "hex");
    serialized.writeUInt8(2, 32); // action
    serialized.writeUInt16BE(0, 33); // set chain to zero to be used for all chains
    serialized.writeUInt32BE(newGuardianSetIndex, 35);
    serialized.writeUInt8(numKeys, 39);
    for (let i = 0; i < numKeys; ++i) {
      const publicKey = publicKeys.at(i);
      if (publicKey == undefined) {
        throw Error("publicKey == undefined");
      }
      serialized.write(publicKey.toString("hex"), 40 + KEY_LENGTH * i, "hex");
    }

    return this.publishMessage(0, serialized, 1, now());
  }
}

export class TokenBridgeGovernanceEmitter extends MockEmitter {
  module: Buffer;

  constructor(emitterAddress: PublicKey) {
    super(emitterAddress.toBuffer().toString("hex"), 1);

    this.module = Buffer.alloc(32);
    this.module.write("TokenBridge", 28);
  }
}

function ethPrivateToPublic(key: string) {
  const ecdsa = new elliptic.ec("secp256k1");
  const publicKey = ecdsa.keyFromPrivate(key).getPublic("hex");
  return keccak256(Buffer.from(publicKey, "hex").subarray(1)).subarray(12);
}

function now() {
  return Math.floor(Date.now() / 1000);
}
