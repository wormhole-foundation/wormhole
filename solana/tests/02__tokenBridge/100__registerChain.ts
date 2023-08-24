import {
  CHAIN_ID_OPTIMISM,
  CHAIN_ID_SOLANA,
  parseVaa,
  tryNativeToHexString,
  tryNativeToUint8Array,
} from "@certusone/wormhole-sdk";
import {
  GovernanceEmitter,
  MockEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  GUARDIAN_KEYS,
  OPTIMISM_TOKEN_BRIDGE_ADDRESS,
  expectIxErr,
  expectIxOk,
  invokeVerifySignaturesAndPostVaa,
  parallelPostVaa,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 2;
const GOVERNANCE_SEQUENCE = 2_010_000;
const GOVERNANCE_MODULE = "000000000000000000000000000000000000000000546f6b656e427269646765";
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

  const wormholeProgram = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  // Test variables.
  const localVariables = new Map<string, any>();

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    it.skip("Invoke `register_chain`", async () => {
      // Fetch default VAA.
      const signedVaa = defaultVaa();

      // Set the message fee for both programs.
      await parallelTxOk(program, forkedProgram, { payer: payer.publicKey }, signedVaa, payer);

      // Check registered emitter account.
      const foreignEmitterData = await tokenBridge.RegisteredEmitter.fromPda(
        connection,
        program.programId,
        CHAIN_ID_OPTIMISM,
        Array.from(tryNativeToUint8Array(OPTIMISM_TOKEN_BRIDGE_ADDRESS, CHAIN_ID_OPTIMISM))
      );
      expect(foreignEmitterData.chain).to.equal(CHAIN_ID_OPTIMISM);
      expect(foreignEmitterData.contract).to.deep.equal(
        Array.from(tryNativeToUint8Array(OPTIMISM_TOKEN_BRIDGE_ADDRESS, CHAIN_ID_OPTIMISM))
      );

      // Save the VAA.
      localVariables.set("signedVaa", signedVaa);
    });
  });

  describe("New Implementation", () => {
    it.skip("Cannot Invoke `register_chain` with Same VAA", async () => {
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

    it.skip("Cannot Invoke `register_chain` with Invalid Emitter Chain ID", async () => {
      const invalidGovernanceChain = 2;
      const sequence = 0;
      const emitterChain = 3;

      // Create a bogus governance VAA.
      const signedVaa = createInvalidRegisterChainVaa(
        GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
        invalidGovernanceChain,
        sequence,
        emitterChain
      );

      // Post the signed Vaa.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      await expectIxErr(
        connection,
        [
          tokenBridge.legacyRegisterChainIx(
            program,
            { payer: payer.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "InvalidGovernanceEmitter"
      );
    });

    it.skip("Cannot Invoke `register_chain` with Invalid Emitter Address", async () => {
      const invalidGovernanceEmitter = tryNativeToHexString(
        OPTIMISM_TOKEN_BRIDGE_ADDRESS,
        CHAIN_ID_OPTIMISM
      );
      const governanceChain = CHAIN_ID_SOLANA;
      const sequence = 1;
      const emitterChain = 4;

      // Create a bogus governance VAA.
      const signedVaa = createInvalidRegisterChainVaa(
        invalidGovernanceEmitter, // Invalid governance emitter.
        governanceChain,
        sequence,
        emitterChain
      );

      // Post the signed Vaa.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      await expectIxErr(
        connection,
        [
          tokenBridge.legacyRegisterChainIx(
            program,
            { payer: payer.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "InvalidGovernanceEmitter"
      );
    });

    it.skip("Cannot Invoke `register_chain` with Invalid Governance Module", async () => {
      const governanceChain = CHAIN_ID_SOLANA;
      const sequence = 2;
      const emitterChain = 5;

      // Create a bogus governance VAA.
      const signedVaa = createInvalidRegisterChainVaa(
        GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"), // Legit governance emitter.
        governanceChain, // Legit governance chain.
        sequence,
        emitterChain,
        Buffer.from("00000000000000000000000000000000000000000000000000000000deadbeef", "hex") // Invalid module.
      );

      // Post the signed Vaa.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      await expectIxErr(
        connection,
        [
          tokenBridge.legacyRegisterChainIx(
            program,
            { payer: payer.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "InvalidGovernanceVaa"
      );
    });

    it.skip("Cannot Invoke `register_chain` with Invalid Governance Action", async () => {
      const governanceChain = CHAIN_ID_SOLANA;
      const sequence = 3;
      const emitterChain = 6;
      const invalidGovernanceAction = 2;

      // Create a bogus governance VAA.
      const signedVaa = createInvalidRegisterChainVaa(
        GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"), // Legit governance emitter.
        governanceChain, // Legit governance chain.
        sequence,
        emitterChain,
        Buffer.from(GOVERNANCE_MODULE, "hex"),
        invalidGovernanceAction
      );

      // Post the signed Vaa.
      await invokeVerifySignaturesAndPostVaa(wormholeProgram, payer, signedVaa);

      await expectIxErr(
        connection,
        [
          tokenBridge.legacyRegisterChainIx(
            program,
            { payer: payer.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "InvalidGovernanceVaa"
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
    OPTIMISM_TOKEN_BRIDGE_ADDRESS
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

function createInvalidRegisterChainVaa(
  emitter: string,
  chain: number,
  sequence: number,
  emitterChain?: number,
  governanceModule?: Buffer,
  governanceAction?: number
): Buffer {
  const mockEmitter = new MockEmitter(emitter, chain, sequence);

  if (emitterChain === undefined) {
    emitterChain = CHAIN_ID_OPTIMISM;
  }

  if (governanceModule === undefined) {
    governanceModule = Buffer.from(
      "000000000000000000000000000000000000000000546f6b656e427269646765",
      "hex"
    );
  }

  if (governanceAction === undefined) {
    governanceAction = 1;
  }

  // Mock register chain payload.
  let payload = Buffer.alloc(69);
  payload.set(governanceModule, 0);
  payload.writeUint8(governanceAction, 32);
  payload.writeUint16BE(0, 33);
  payload.writeUInt16BE(emitterChain, 35); // Bogus chain ID.
  payload.set(
    Buffer.from(tryNativeToUint8Array(OPTIMISM_TOKEN_BRIDGE_ADDRESS, CHAIN_ID_OPTIMISM)),
    37
  );

  // Vaa info.
  const published = mockEmitter.publishMessage(
    69, // Nonce.
    payload,
    1 // Finality.
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
