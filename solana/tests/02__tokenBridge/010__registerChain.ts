import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GUARDIAN_KEYS,
  expectIxErr,
  expectIxOk,
  parallelPostVaa,
} from "../helpers";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 2;
const GOVERNANCE_SEQUENCE = 2_010_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Token Bridge -- Legacy Instruction: Register Chain", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());
  // Test variables.
  const localVariables = new Map<string, any>();

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    it("Invoke `register_chain`", async () => {
      // Fetch default VAA.
      const signedVaa = defaultVaa();

      // Set the message fee for both programs.
      await parallelTxOk(program, forkedProgram, { payer: payer.publicKey }, signedVaa, payer);

      // TODO: check registered emitter

      // Save the VAA.
      localVariables.set("signedVaa", signedVaa);
    });
  });

  describe("New Implementation", () => {
    it("Cannot Invoke `register_chain` with Same VAA", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa");

      await expectIxErr(
        connection,
        [
          tokenBridge.legacyRegisterChainIx(
            program,
            {
              payer: payer.publicKey,
            },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "already in use"
      );
    });
  });
});

function defaultVaa(): Buffer {
  // Vaa info.
  const timestamp = 12345678;
  const chain = 2;
  const published = governance.publishTokenBridgeRegisterChain(
    timestamp,
    chain,
    ETHEREUM_TOKEN_BRIDGE_ADDRESS
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxOk(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  accounts: tokenBridge.LegacyRegisterChainContext,
  signedVaa: Buffer,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Post the VAAs.
  await parallelPostVaa(connection, payer, signedVaa);

  // Parse the VAA.
  const parsedVaa = parseVaa(signedVaa);

  // Create the set fee instructions.
  const ix = tokenBridge.legacyRegisterChainIx(program, accounts, parsedVaa);
  const forkedIx = tokenBridge.legacyRegisterChainIx(forkedProgram, accounts, parsedVaa);

  return expectIxOk(connection, [ix, forkedIx], [payer]);
}
