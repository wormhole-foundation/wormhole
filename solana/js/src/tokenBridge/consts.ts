import { PublicKey } from "@solana/web3.js";

export type ProgramId =
  | "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb" // mainnet
  | "DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe" // testnet
  | "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE" // devnet
  | "bPPNmBhmHfkEFJmNKKCvwc1tPqBjzPDRwCw3yQYYXQa"; // localnet

export const TOKEN_BRIDGE_PROGRAM_ID = new PublicKey(
  "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb"
);
