import { PublicKey } from "@solana/web3.js";

export type ProgramId =
  | "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth" // mainnet
  | "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5" // testnet
  | "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o" // devnet
  | "agnnozV7x6ffAhi8xVhBd5dShfLnuUKKPEMX1tJ1nDC"; // localnet

export const CORE_BRIDGE_PROGRAM_ID = new PublicKey(
  "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
);
