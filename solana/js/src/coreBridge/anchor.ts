import { Program } from "@coral-xyz/anchor";
import { Connection, PublicKey } from "@solana/web3.js";
import CoreBridgeIdl from "../idl/solana_wormhole_core_bridge.json";
import { SolanaWormholeCoreBridge } from "../types/solana_wormhole_core_bridge";
import { CORE_BRIDGE_PROGRAM_ID, ProgramId } from "./consts";

export function coreBridgeProgram(
  connection: Connection,
  programId?: ProgramId
): Program<SolanaWormholeCoreBridge> {
  return new Program<SolanaWormholeCoreBridge>(
    CoreBridgeIdl as SolanaWormholeCoreBridge,
    programId === undefined ? CORE_BRIDGE_PROGRAM_ID : new PublicKey(programId),
    { connection }
  );
}
