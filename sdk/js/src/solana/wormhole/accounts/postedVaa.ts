import { Commitment, Connection, PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { deriveAddress, getAccountData } from "../../utils";
import { MessageData } from "../message";

export class PostedVaaData {
  message: MessageData;

  constructor(message: MessageData) {
    this.message = message;
  }

  static deserialize(data: Buffer) {
    return new PostedVaaData(MessageData.deserialize(data.subarray(3)));
  }
}

export function postedVaaKey(wormholeProgramId: PublicKeyInitData, hash: Buffer): PublicKey {
  return deriveAddress([Buffer.from("PostedVAA"), hash], wormholeProgramId);
}

export async function getPostedVaa(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  hash: Buffer,
  commitment?: Commitment
): Promise<PostedVaaData> {
  return connection
    .getAccountInfo(postedVaaKey(wormholeProgramId, hash), commitment)
    .then((info) => PostedVaaData.deserialize(getAccountData(info)));
}
