import { PublicKey } from "@solana/web3.js";

export type ProgramId =
  | "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth" // mainnet
  | "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5" // testnet
  | "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"; // localnet

export const GOVERNANCE_EMITTER_ADDRESS = new PublicKey("11111111111111111111111111111115");
