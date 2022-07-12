import {
  Connection,
  PublicKey,
  Commitment,
  PublicKeyInitData,
} from "@solana/web3.js";
import { serializeUint16 } from "byteify";
import {
  ChainId,
  CHAIN_ID_SOLANA,
  tryNativeToUint8Array,
} from "../../../utils";
import { deriveAddress, getAccountData } from "../../utils";

export function deriveEndpointKey(
  tokenBridgeProgramId: PublicKeyInitData,
  emitterChain: number | ChainId,
  emitterAddress: Buffer | Uint8Array | string
): PublicKey {
  if (emitterChain == CHAIN_ID_SOLANA) {
    throw new Error(
      "emitterChain == CHAIN_ID_SOLANA cannot exist as foreign token bridge emitter"
    );
  }
  if (typeof emitterAddress == "string") {
    emitterAddress = tryNativeToUint8Array(
      emitterAddress,
      emitterChain as ChainId
    );
  }
  return deriveAddress(
    [serializeUint16(emitterChain as number), emitterAddress],
    tokenBridgeProgramId
  );
}

export async function getEndpointRegistration(
  connection: Connection,
  endpointKey: PublicKeyInitData,
  commitment?: Commitment
): Promise<EndpointRegistration> {
  return connection
    .getAccountInfo(new PublicKey(endpointKey), commitment)
    .then((info) => EndpointRegistration.deserialize(getAccountData(info)));
}

export class EndpointRegistration {
  // pub struct EndpointRegistration {
  //     pub chain: ChainID,
  //     pub contract: Address,
  // }

  chain: ChainId;
  contract: Buffer;

  constructor(chain: number, contract: Buffer) {
    this.chain = chain as ChainId;
    this.contract = contract;
  }

  static deserialize(data: Buffer): EndpointRegistration {
    if (data.length != 34) {
      throw new Error("data.length != 34");
    }
    const chain = data.readUInt16LE(0);
    const contract = data.subarray(2, 34);
    return new EndpointRegistration(chain, contract);
  }
}
