import { BN } from "@coral-xyz/anchor";
import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
  PublicKeyInitData,
} from "@solana/web3.js";

// TODO: Write something about how this account can be used for any program
// that integrates with the core bridge.
export class Claim {
  static address(
    programId: PublicKeyInitData,
    chain: number,
    address: number[],
    sequence: BN
  ): PublicKey {
    const chainBuf = Buffer.alloc(2);
    chainBuf.writeUInt16BE(chain);

    const addressBuf = Buffer.alloc(32);
    addressBuf.set(address);

    const sequenceBuf = Buffer.alloc(8);
    sequenceBuf.writeBigUInt64BE(BigInt(sequence.toString()));

    return PublicKey.findProgramAddressSync(
      [chainBuf, addressBuf, sequenceBuf],
      new PublicKey(programId)
    )[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): Claim {
    throw new Error("not implemented");
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<Claim> {
    throw new Error("not implemented");
  }

  static deserialize(data: Buffer): Claim {
    throw new Error("not implemented");
  }
}
