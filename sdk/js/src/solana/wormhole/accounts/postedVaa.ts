import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
} from "@solana/web3.js";
import { deriveAddress, getAccountData } from "../../utils";
import { MessageData } from "../message";

export class PostedMessageData {
  message: MessageData;

  constructor(message: MessageData) {
    this.message = message;
  }

  static deserialize(data: Buffer) {
    return new PostedMessageData(MessageData.deserialize(data.subarray(3)));
  }
}

export class PostedVaaData extends PostedMessageData {}

export function derivePostedVaaKey(
  wormholeProgramId: PublicKeyInitData,
  hash: Buffer
): PublicKey {
  return deriveAddress([Buffer.from("PostedVAA"), hash], wormholeProgramId);
}

export async function getPostedVaa(
  connection: Connection,
  wormholeProgramId: PublicKeyInitData,
  hash: Buffer,
  commitment?: Commitment
): Promise<PostedVaaData> {
  return connection
    .getAccountInfo(derivePostedVaaKey(wormholeProgramId, hash), commitment)
    .then((info) => PostedVaaData.deserialize(getAccountData(info)));
}

export async function getPostedMessage(
  connection: Connection,
  messageKey: PublicKeyInitData,
  commitment?: Commitment
): Promise<PostedMessageData> {
  return connection
    .getAccountInfo(new PublicKey(messageKey), commitment)
    .then((info) => PostedMessageData.deserialize(getAccountData(info)));
}
