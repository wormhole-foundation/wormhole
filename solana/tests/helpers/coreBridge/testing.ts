import { BridgeProgramData, CoreBridgeProgram } from ".";
import { expectDeepEqual } from "../utils";

export async function expectEqualBridgeAccounts(
  program: CoreBridgeProgram,
  forkedProgram: CoreBridgeProgram
) {
  const connection = program.provider.connection;

  const [bridgeData, forkBridgeData] = await Promise.all([
    BridgeProgramData.fromPda(connection, program.programId),
    BridgeProgramData.fromPda(connection, forkedProgram.programId),
  ]);
  expectDeepEqual(bridgeData, forkBridgeData);
}
