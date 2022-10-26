import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
  TransactionInstruction,
} from "@solana/web3.js";
import { Program, Provider } from "@project-serum/anchor";
import { createReadOnlyProvider } from "../utils";
import { WormholeCoder } from "./coder";
import { Wormhole } from "../types/wormhole";

import IDL from "../../anchor-idl/wormhole.json";

export function createWormholeProgramInterface(
  programId: PublicKeyInitData,
  provider?: Provider
): Program<Wormhole> {
  return new Program<Wormhole>(
    IDL as Wormhole,
    new PublicKey(programId),
    provider === undefined ? ({ connection: null } as any) : provider,
    coder()
  );
}

export function createReadOnlyWormholeProgramInterface(
  programId: PublicKeyInitData,
  connection?: Connection
): Program<Wormhole> {
  return createWormholeProgramInterface(
    programId,
    createReadOnlyProvider(connection)
  );
}

export function coder(): WormholeCoder {
  return new WormholeCoder(IDL as Wormhole);
}
