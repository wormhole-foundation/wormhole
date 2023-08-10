import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";
import { PostedMessageV1 } from "./PostedMessageV1";

export class PostedMessageV1Unreliable extends PostedMessageV1 {
  static fromAccountInfo(info: AccountInfo<Buffer>): PostedMessageV1Unreliable {
    const data = info.data;
    const discriminator = data.subarray(0, 4);
    if (!discriminator.equals(Buffer.from([109, 115, 117, 0]))) {
      throw new Error(`Invalid discriminator: ${discriminator}`);
    }
    return PostedMessageV1.deserialize(data.subarray(4)) as PostedMessageV1Unreliable;
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<PostedMessageV1Unreliable> {
    const accountInfo = await connection.getAccountInfo(address, commitmentOrConfig);
    if (accountInfo == null) {
      throw new Error(`Unable to find PostedMessageV1Unreliable account at ${address}`);
    }
    return PostedMessageV1Unreliable.fromAccountInfo(accountInfo);
  }
}
