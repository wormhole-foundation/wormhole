import { BN } from "@coral-xyz/anchor";
import {
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
} from "@solana/web3.js";
import { BPF_LOADER_UPGRADEABLE_PROGRAM_ID, ProgramData } from "../../native";
import { coreBridgeProgram } from "../anchor";
import { ProgramId } from "../consts";
import {
  BridgeProgramData,
  FeeCollector,
  GuardianPubkey,
  GuardianSet,
} from "../state";
import { getProgramPubkey, upgradeAuthority } from "../utils";

export class InitializeContext {
  deployer: PublicKey;
  bridge: PublicKey;
  guardianSet: PublicKey;
  feeCollector: PublicKey;
  upgradeAuthority: PublicKey;
  programData: PublicKey;
  bpfLoaderUpgradeableProgram: PublicKey;
  systemProgram: PublicKey;

  private constructor(programId: ProgramId, deployer: PublicKeyInitData) {
    this.deployer = new PublicKey(deployer);
    this.bridge = BridgeProgramData.address(programId);
    this.guardianSet = GuardianSet.address(programId, 0);
    this.feeCollector = FeeCollector.address(programId);
    this.upgradeAuthority = upgradeAuthority(programId);
    this.programData = ProgramData.address(getProgramPubkey(programId));
    this.bpfLoaderUpgradeableProgram = BPF_LOADER_UPGRADEABLE_PROGRAM_ID;
    this.systemProgram = SystemProgram.programId;
  }

  static new(programId: ProgramId, deployer: PublicKeyInitData) {
    return new InitializeContext(programId, deployer);
  }

  static instruction(
    connection: Connection,
    programId: ProgramId,
    deployer: PublicKeyInitData,
    args: InitializeArgs
  ) {
    return initializeIx(
      connection,
      programId,
      InitializeContext.new(programId, deployer),
      args
    );
  }
}

export type InitializeArgs = {
  guardianSetTtlSeconds: number;
  feeLamports: BN;
  initialGuardians: GuardianPubkey[];
};

export async function initializeIx(
  connection: Connection,
  programId: ProgramId,
  accounts: InitializeContext,
  args: InitializeArgs
) {
  const program = coreBridgeProgram(connection, programId);
  return program.methods.initialize(args).accounts(accounts).instruction();
}
