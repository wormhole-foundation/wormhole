import { Program } from "@coral-xyz/anchor";
import { Connection, PublicKey } from "@solana/web3.js";
import MockCpiIdl from "../../../target/idl/wormhole_mock_cpi_solana.json";
import { WormholeMockCpiSolana } from "../../../target/types/wormhole_mock_cpi_solana";
import * as coreBridge from "../coreBridge";
import * as tokenBridge from "../tokenBridge";
import { ProgramId } from "./consts";

export * from "./consts";
// export * from "./instructions";
// export * from "./legacy";
// export * from "./testing";

export type MockCpiProgram = Program<WormholeMockCpiSolana>;

export function getProgramId(programId?: ProgramId): PublicKey {
  return new PublicKey(
    programId === undefined
      ? "MockCpi696969696969696969696969696969696969" // localnet
      : programId
  );
}

export function getAnchorProgram(connection: Connection, programId: PublicKey): MockCpiProgram {
  return new Program<WormholeMockCpiSolana>(MockCpiIdl as any, programId, { connection });
}

export function localnet(): PublicKey {
  return getProgramId("MockCpi696969696969696969696969696969696969");
}

export function coreBridgeProgramId(program: MockCpiProgram): PublicKey {
  return coreBridge.localnet();
}

export function getCoreBridgeProgram(program: MockCpiProgram): coreBridge.CoreBridgeProgram {
  return coreBridge.getAnchorProgram(program.provider.connection, coreBridgeProgramId(program));
}

export function tokenBridgeProgramId(program: MockCpiProgram): PublicKey {
  return tokenBridge.localnet();
}

export function getTokenBridgeProgram(program: MockCpiProgram): tokenBridge.TokenBridgeProgram {
  return tokenBridge.getAnchorProgram(program.provider.connection, tokenBridgeProgramId(program));
}
