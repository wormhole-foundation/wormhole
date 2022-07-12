import { Connection, PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { Program, Provider } from "@project-serum/anchor";
import { createReadOnlyProvider } from "../utils";
import { NftBridgeCoder } from "./coder";
import { NftBridge } from "../types/nftBridge";

import IDL from "../../anchor-idl/token_bridge.json";

export function createNftBridgeProgramInterface(
  programId: PublicKeyInitData,
  provider?: Provider
): Program<NftBridge> {
  return new Program<NftBridge>(
    IDL as NftBridge,
    new PublicKey(programId),
    provider == undefined ? ({ connection: null } as any) : provider,
    coder()
  );
}

export function createReadOnlyNftBridgeProgramInterface(
  programId: PublicKeyInitData,
  connection?: Connection
): Program<NftBridge> {
  return createNftBridgeProgramInterface(
    programId,
    createReadOnlyProvider(connection)
  );
}

export function coder(): NftBridgeCoder {
  return new NftBridgeCoder(IDL as NftBridge);
}
