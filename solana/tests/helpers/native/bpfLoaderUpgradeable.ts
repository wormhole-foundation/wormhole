import { BN } from "@coral-xyz/anchor";
import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
  PublicKeyInitData,
} from "@solana/web3.js";

export const BPF_LOADER_UPGRADEABLE_PROGRAM_ID = new PublicKey(
  "BPFLoaderUpgradeab1e11111111111111111111111"
);

export type BpfUninitialized = { type: "Uninitialized" };

export type BpfBuffer = {
  type: "Buffer";
  authorityAddress: PublicKey | null;
};

export type BpfProgram = {
  type: "Program";
  programdataAddress: PublicKey;
};

export type BpfProgramData = {
  type: "ProgramData";
  slot: BN;
  upgradeAuthorityAddress: PublicKey | null;
};

export type UpgradableLoaderState =
  | BpfUninitialized
  | BpfBuffer
  | BpfProgram
  | BpfProgramData;

export class ProgramData {
  static address(programId: PublicKeyInitData): PublicKey {
    const [addr] = PublicKey.findProgramAddressSync(
      [new PublicKey(programId).toBuffer()],
      BPF_LOADER_UPGRADEABLE_PROGRAM_ID
    );
    return addr;
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): ProgramData {
    return ProgramData.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<ProgramData> {
    const accountInfo = await connection.getAccountInfo(
      address,
      commitmentOrConfig
    );
    if (accountInfo == null) {
      throw new Error(
        `Unable to find BPFLoaderUpgradeable Program Data at ${address}`
      );
    }
    return ProgramData.fromAccountInfo(accountInfo);
  }

  static deserialize(data: Buffer): ProgramData {
    switch (data.readUInt32LE(0)) {
      case 0: {
        return { type: "Uninitialized" };
      }
      case 1: {
        return {
          type: "Buffer",
          authorityAddress:
            data[4] === 0 ? null : new PublicKey(data.subarray(5, 37)),
        };
      }
      case 2: {
        throw new Error("Not implemented yet.");
      }
      case 3: {
        const slot = new BN(data.readBigUInt64LE(4).toString());
        return {
          type: "ProgramData",
          slot,
          upgradeAuthorityAddress:
            data[12] === 0 ? null : new PublicKey(data.subarray(13, 45)),
        };
      }
      default: {
        throw new Error("Invalid program data.");
      }
    }
  }
}
