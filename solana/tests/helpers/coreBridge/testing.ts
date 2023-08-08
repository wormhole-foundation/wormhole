import { BridgeProgramData, CoreBridgeProgram } from ".";
import { expectDeepEqual } from "../utils";
import * as anchor from "@coral-xyz/anchor";
import * as coreBridge from "../coreBridge";

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

export async function expectEqualMessageAccounts(
  program: CoreBridgeProgram,
  messageSigner: anchor.web3.Keypair,
  forkedMessageSigner: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  const [messageData, forkedMessageData] = await Promise.all([
    coreBridge.PostedMessageV1.fromAccountAddress(connection, messageSigner.publicKey),
    coreBridge.PostedMessageV1.fromAccountAddress(connection, forkedMessageSigner.publicKey),
  ]);
  expectDeepEqual(messageData, forkedMessageData);
}
