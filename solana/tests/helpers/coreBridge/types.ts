import { Commitment } from "@solana/web3.js";

export enum MessageCommitment {
  Confirmed,
  Finalized,
}

export function toLegacyCommitment(commitment: Commitment): MessageCommitment {
  switch (commitment) {
    case "confirmed":
      return MessageCommitment.Confirmed;
    case "finalized":
      return MessageCommitment.Finalized;
    default:
      throw new Error(`Invalid commitment: ${commitment}`);
  }
}

export function toMessageCommitment(commitment: Commitment) {
  switch (commitment) {
    case "confirmed":
      return { confirmed: {} };
    case "finalized":
      return { finalized: {} };
    default:
      throw new Error(`Invalid commitment: ${commitment}`);
  }
}

export function toConsistencyLevel(commitment: Commitment) {
  switch (commitment) {
    case "confirmed":
      return 1;
    case "finalized":
      return 32;
    default:
      throw new Error(`Invalid commitment: ${commitment}`);
  }
}

export function fromConsistencyLevel(consistencyLevel: number): Commitment {
  switch (consistencyLevel) {
    case 1:
      return "confirmed";
    case 32:
      return "finalized";
    default:
      throw new Error(`Invalid consistency level: ${consistencyLevel}`);
  }
}
