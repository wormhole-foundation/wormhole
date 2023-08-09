import { Program } from "@coral-xyz/anchor";
import { Connection, PublicKey } from "@solana/web3.js";
import TokenBridgeIdl from "../../../target/idl/wormhole_token_bridge_solana.json";
import { WormholeTokenBridgeSolana } from "../../../target/types/wormhole_token_bridge_solana";
import { ProgramId } from "./consts";

export * from "./consts";
// export * from "./instructions";
export * from "./legacy";
// export * from "./state";
// export * from "./testing";

export type TokenBridgeProgram = Program<WormholeTokenBridgeSolana>;

export function getProgramId(programId?: ProgramId): PublicKey {
  return new PublicKey(
    programId === undefined
      ? "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb" // mainnet
      : programId
  );
}

export function getAnchorProgram(
  connection: Connection,
  programId: PublicKey
): TokenBridgeProgram {
  return new Program<WormholeTokenBridgeSolana>(
    TokenBridgeIdl as any,
    programId,
    { connection }
  );
}
