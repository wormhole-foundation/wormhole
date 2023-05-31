import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";
import { ProgramId } from "../consts";
import { getProgramPubkey } from "../utils";

export class PostedVaaV1 {
  static address(programId: ProgramId, hash: number[]): PublicKey {
    return PublicKey.findProgramAddressSync(
      [Buffer.from("PostedVAA"), Buffer.from(hash)],
      getProgramPubkey(programId)
    )[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): PostedVaaV1 {
    const data = info.data;
    const discriminator = data.subarray(0, 4);
    console.log({ discriminator });
    if (!discriminator.equals(Buffer.from([118, 97, 97, 1]))) {
      throw new Error(`Invalid discriminator: ${discriminator}`);
    }
    return PostedVaaV1.deserialize(data.subarray(3));
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<PostedVaaV1> {
    const accountInfo = await connection.getAccountInfo(
      address,
      commitmentOrConfig
    );
    if (accountInfo == null) {
      throw new Error(`Unable to find PostedVaaV1 account at ${address}`);
    }
    return PostedVaaV1.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: ProgramId,
    hash: number[]
  ): Promise<PostedVaaV1> {
    return PostedVaaV1.fromAccountAddress(
      connection,
      PostedVaaV1.address(programId, hash)
    );
  }

  static deserialize(data: Buffer): PostedVaaV1 {
    throw new Error("not implemented");
  }
}
