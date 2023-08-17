import { Commitment, Connection, PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram, getAnchorProgram } from "..";

export enum ProcessingStatus {
  Unset = 0,
  Writing = 1,
  Verified = 2,
}

export enum VaaVersion {
  Unset = 0,
  V1 = 1,
}

export class EncodedVaa {
  status: ProcessingStatus;
  writeAuthority: PublicKey;
  version: VaaVersion;
  buf: Buffer;

  private constructor(
    status: ProcessingStatus,
    writeAuthority: PublicKey,
    version: VaaVersion,
    buf: Buffer
  ) {
    this.status = status;
    this.writeAuthority = writeAuthority;
    this.version = version;
    this.buf = buf;
  }

  static discriminator() {
    return Uint8Array.from([226, 101, 163, 4, 133, 160, 84, 245]);
  }

  static async fetch(program: CoreBridgeProgram, address: PublicKey, commitment?: Commitment) {
    const {
      header: { status: processingStatus, writeAuthority, version: vaaVersion },
      buf,
    } = await program.account.encodedVaa.fetch(address, commitment);

    const status = (() => {
      if (processingStatus.unset !== undefined) {
        return ProcessingStatus.Unset;
      } else if (processingStatus.writing !== undefined) {
        return ProcessingStatus.Writing;
      } else if (processingStatus.verified !== undefined) {
        return ProcessingStatus.Verified;
      } else {
        throw new Error("Invalid processing status");
      }
    })();

    const version = (() => {
      if (vaaVersion.unset !== undefined) {
        return VaaVersion.Unset;
      } else if (vaaVersion.v1 !== undefined) {
        return VaaVersion.V1;
      } else {
        throw new Error("Invalid processing status");
      }
    })();

    return new EncodedVaa(status, writeAuthority, version, buf);
  }
}
