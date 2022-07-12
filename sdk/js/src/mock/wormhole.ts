import { keccak256 } from "../utils";
import { ethPrivateToPublic, ethSignWithPrivate } from "./misc";

const SIGNATURE_PAYLOAD_LEN = 66;

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

    for (let i = 0; i < numSigners; ++i) {
      const signer = signers.at(i);
      if (signer == undefined) {
        throw Error("signer == undefined");
      }
      const signature = ethSignWithPrivate(signer.key, hash);

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
      signedVaa.writeUInt8(signature.recoveryParam!, start + 65);
    }

    return signedVaa;
  }
}

export class MockEmitter {
  chain: number;
  address: Buffer;

  sequence: number;

  constructor(emitterAddress: string, chain: number, startSequence?: number) {
    this.chain = chain;
    const address = Buffer.from(emitterAddress, "hex");
    if (address.length != 32) {
      throw Error("emitterAddress.length != 32");
    }
    this.address = address;

    this.sequence = startSequence == undefined ? 0 : startSequence;
  }

  publishMessage(
    nonce: number,
    payload: Buffer,
    consistencyLevel: number,
    timestamp?: number,
    uptickSequence: boolean = true
  ) {
    if (uptickSequence) {
      ++this.sequence;
    }

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
