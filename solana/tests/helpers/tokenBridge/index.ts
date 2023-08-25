import { Program } from "@coral-xyz/anchor";
import { Connection, PublicKey } from "@solana/web3.js";
import TokenBridgeIdl from "../../../target/idl/wormhole_token_bridge_solana.json";
import { WormholeTokenBridgeSolana } from "../../../target/types/wormhole_token_bridge_solana";
import * as coreBridge from "../coreBridge";
import { ProgramId } from "./consts";

export * from "./consts";
export * from "./instructions";
export * from "./legacy";
export * from "./testing";

export type TokenBridgeProgram = Program<WormholeTokenBridgeSolana>;

export function getProgramId(programId?: ProgramId): PublicKey {
  return new PublicKey(
    programId === undefined
      ? "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb" // mainnet
      : programId
  );
}

export function getAnchorProgram(connection: Connection, programId: PublicKey): TokenBridgeProgram {
  return new Program<WormholeTokenBridgeSolana>(TokenBridgeIdl as any, programId, { connection });
}

export function mainnet(): PublicKey {
  return getProgramId();
}

export function localnet(): PublicKey {
  return getProgramId("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE");
}

export function coreBridgeProgramId(program: TokenBridgeProgram): PublicKey {
  switch (program.programId.toString() as ProgramId) {
    case "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb": {
      return coreBridge.mainnet();
    }
    case "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE": {
      return coreBridge.localnet();
    }
    default: {
      throw new Error("unsupported");
    }
  }
}

export function getCoreBridgeProgram(program: TokenBridgeProgram): coreBridge.CoreBridgeProgram {
  return coreBridge.getAnchorProgram(program.provider.connection, coreBridgeProgramId(program));
}
