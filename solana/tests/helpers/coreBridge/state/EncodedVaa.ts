import { Commitment, Connection, PublicKey } from "@solana/web3.js";
import { getAnchorProgram } from "..";

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
  bytes: Buffer;

  private constructor(
    status: ProcessingStatus,
    writeAuthority: PublicKey,
    version: VaaVersion,
    bytes: Buffer
  ) {
    this.status = status;
    this.writeAuthority = writeAuthority;
    this.version = version;
    this.bytes = bytes;
  }

  discriminator() {
    return Uint8Array.from([226, 101, 163, 4, 133, 160, 84, 245]);
  }

  static async fromAccountAddress(
    connection: Connection,
    programId: PublicKey,
    address: PublicKey,
    commitment?: Commitment
  ) {
    const program = getAnchorProgram(connection, programId);
    const {
      header: { status: processingStatus, writeAuthority, version: vaaVersion },
      bytes,
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

    return new EncodedVaa(status, writeAuthority, version, bytes);
  }
}
