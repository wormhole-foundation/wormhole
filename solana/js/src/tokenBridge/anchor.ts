import { Program } from "@coral-xyz/anchor";
import { Connection, PublicKey } from "@solana/web3.js";
import TokenBridgeIdl from "../idl/solana_wormhole_token_bridge.json";
import { SolanaWormholeTokenBridge } from "../types/solana_wormhole_token_bridge";
import { TOKEN_BRIDGE_PROGRAM_ID, ProgramId } from "./consts";

export function coreBridgeProgram(
  connection: Connection,
  programId?: ProgramId
): Program<SolanaWormholeTokenBridge> {
  return new Program<SolanaWormholeTokenBridge>(
    TokenBridgeIdl as SolanaWormholeTokenBridge,
    programId === undefined
      ? TOKEN_BRIDGE_PROGRAM_ID
      : new PublicKey(programId),
    { connection }
  );
}
