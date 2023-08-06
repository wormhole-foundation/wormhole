import {
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
} from "@solana/web3.js";
import { BPF_LOADER_UPGRADEABLE_PROGRAM_ID, ProgramData } from "../../native";
import { coreBridgeProgram } from "../anchor";
import { ProgramId } from "../consts";
import { Config } from "../state";
import { getProgramPubkey, upgradeAuthority } from "../utils";

export class InitializeContext {
  deployer: PublicKey;
  config: PublicKey;
  upgradeAuthority: PublicKey;
  programData: PublicKey;
  bpfLoaderUpgradeableProgram: PublicKey;
  systemProgram: PublicKey;
  coreBridgeProgram: PublicKey;

  private constructor(
    programId: ProgramId,
    deployer: PublicKeyInitData,
    coreBridgeProgram: PublicKeyInitData
  ) {
    this.deployer = new PublicKey(deployer);
    this.config = Config.address(programId);
    this.upgradeAuthority = upgradeAuthority(programId);
    this.programData = ProgramData.address(getProgramPubkey(programId));
    this.bpfLoaderUpgradeableProgram = BPF_LOADER_UPGRADEABLE_PROGRAM_ID;
    this.systemProgram = SystemProgram.programId;
    this.coreBridgeProgram = new PublicKey(coreBridgeProgram);
  }

  static new(
    programId: ProgramId,
    deployer: PublicKeyInitData,
    coreBridgeProgram: PublicKeyInitData
  ) {
    return new InitializeContext(programId, deployer, coreBridgeProgram);
  }

  static instruction(
    connection: Connection,
    programId: ProgramId,
    deployer: PublicKeyInitData,
    coreBridgeProgram: PublicKeyInitData
  ) {
    return initializeIx(
      connection,
      programId,
      InitializeContext.new(programId, deployer, coreBridgeProgram)
    );
  }
}

export async function initializeIx(
  connection: Connection,
  programId: ProgramId,
  accounts: InitializeContext
) {
  const program = coreBridgeProgram(connection, programId);
  return program.methods.initialize().accounts(accounts).instruction();
}
