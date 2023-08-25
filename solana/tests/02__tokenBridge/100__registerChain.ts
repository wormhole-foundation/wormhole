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
  createAccountIx,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
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

describe("Token Bridge -- Instruction: Register Chain", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());
  const wormholeProgram = tokenBridge.getCoreBridgeProgram(program);

  // Test variables.
  const localVariables = new Map<string, any>();

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    it("Cannot Invoke Legacy `register_chain`", async () => {
      // Fetch default VAA.
      const signedVaa = defaultVaa();

      const ix = tokenBridge.legacyRegisterChainIx(
        program,
        { payer: payer.publicKey },
        parseVaa(signedVaa)
      );

      // The legacy instruction is deprecated and should fail.
      await expectIxErr(connection, [ix], [payer], "Deprecated");
    });

    it("Invoke `register_chain`", async () => {
      // Fetch default VAA.
      const signedVaa = defaultVaa();

      const encodedVaa = await initAndProcessEncodedVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      const ix = await tokenBridge.registerChainIx(program, {
        payer: payer.publicKey,
        vaa: encodedVaa,
      });

      await expectIxOk(connection, [ix], [payer]);

      const registered = tokenBridge.RegisteredEmitter.address(
        program.programId,
        CHAIN_ID_OPTIMISM
      );
      const legacyRegistered = tokenBridge.RegisteredEmitter.address(
        program.programId,
        CHAIN_ID_OPTIMISM,
        Array.from(tryNativeToUint8Array(OPTIMISM_TOKEN_BRIDGE_ADDRESS, CHAIN_ID_OPTIMISM))
      );
      expect(registered.toString()).not.equal(legacyRegistered.toString());

      const registeredData = await tokenBridge.RegisteredEmitter.fromAccountAddress(
        connection,
        registered
      );
      const legacyRegisteredData = await tokenBridge.RegisteredEmitter.fromAccountAddress(
        connection,
        legacyRegistered
      );
      expectDeepEqual(registeredData, legacyRegisteredData);

      const encodedVaaInfo = await connection.getAccountInfo(encodedVaa);
      expect(encodedVaaInfo).is.null;

      // Save for later.
      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `register_chain` with Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa") as Buffer;

      const encodedVaa = await initAndProcessEncodedVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      await expectIxErr(
        connection,
        [
          await tokenBridge.registerChainIx(program, {
            payer: payer.publicKey,
            vaa: encodedVaa,
          }),
        ],
        [payer],
        "already in use"
      );
    });

    it("Cannot Invoke `register_chain` with Invalid Emitter Chain ID", async () => {
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
      const encodedVaa = await initAndProcessEncodedVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      await expectIxErr(
        connection,
        [
          await tokenBridge.registerChainIx(program, {
            payer: payer.publicKey,
            vaa: encodedVaa,
          }),
        ],
        [payer],
        "InvalidGovernanceEmitter"
      );
    });

    it("Cannot Invoke `register_chain` with Invalid Emitter Address", async () => {
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
      const encodedVaa = await initAndProcessEncodedVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      await expectIxErr(
        connection,
        [
          await tokenBridge.registerChainIx(program, {
            payer: payer.publicKey,
            vaa: encodedVaa,
          }),
        ],
        [payer],
        "InvalidGovernanceEmitter"
      );
    });

    it("Cannot Invoke `register_chain` with Invalid Governance Module", async () => {
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
      const encodedVaa = await initAndProcessEncodedVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      await expectIxErr(
        connection,
        [
          await tokenBridge.registerChainIx(program, {
            payer: payer.publicKey,
            vaa: encodedVaa,
          }),
        ],
        [payer],
        "InvalidGovernanceVaa"
      );
    });

    it("Cannot Invoke `register_chain` with Invalid Governance Action", async () => {
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
      const encodedVaa = await initAndProcessEncodedVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      await expectIxErr(
        connection,
        [
          await tokenBridge.registerChainIx(program, {
            payer: payer.publicKey,
            vaa: encodedVaa,
          }),
        ],
        [payer],
        "InvalidGovernanceVaa"
      );
    });
  });
});

function defaultVaa(chain?: number, contract?: string): Buffer {
  // Vaa info.
  const timestamp = 12345678;

  if (chain === undefined) {
    chain = CHAIN_ID_OPTIMISM;
  }

  if (contract === undefined) {
    contract = OPTIMISM_TOKEN_BRIDGE_ADDRESS;
  }

  const published = governance.publishTokenBridgeRegisterChain(timestamp, chain, contract);
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

async function initAndProcessEncodedVaa(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  signedVaa: Buffer
) {
  const connection = program.provider.connection;

  const vaaLen = signedVaa.length;

  const encodedVaa = anchor.web3.Keypair.generate();
  const createIx = await createAccountIx(
    program.provider.connection,
    program.programId,
    payer,
    encodedVaa,
    46 + vaaLen
  );

  const initIx = await coreBridge.initEncodedVaaIx(program, {
    writeAuthority: payer.publicKey,
    encodedVaa: encodedVaa.publicKey,
  });

  const endAfterInit = 840;
  const firstProcessIx = await coreBridge.processEncodedVaaIx(
    program,
    {
      writeAuthority: payer.publicKey,
      encodedVaa: encodedVaa.publicKey,
      guardianSet: null,
    },
    { write: { index: 0, data: signedVaa.subarray(0, endAfterInit) } }
  );

  if (vaaLen > endAfterInit) {
    await expectIxOk(
      program.provider.connection,
      [createIx, initIx, firstProcessIx],
      [payer, encodedVaa]
    );

    const promises: Promise<string>[] = [];
    const chunkSize = 912;
    for (let start = endAfterInit; start < vaaLen; start += chunkSize) {
      const end = Math.min(start + chunkSize, vaaLen);

      const writeIx = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa: encodedVaa.publicKey,
          guardianSet: null,
        },
        { write: { index: start, data: signedVaa.subarray(start, end) } }
      );

      if (end === vaaLen) {
        const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });
        const verifyIx = await coreBridge.processEncodedVaaIx(
          program,
          {
            writeAuthority: payer.publicKey,
            encodedVaa: encodedVaa.publicKey,
            guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
          },
          { verifySignaturesV1: {} }
        );
        promises.push(expectIxOk(connection, [computeIx, writeIx, verifyIx], [payer]));
      } else {
        promises.push(expectIxOk(connection, [writeIx], [payer]));
      }
    }

    const lastPromise = promises.pop();

    if (promises.length > 0) {
      await Promise.all(promises);
    }

    await lastPromise;
  } else {
    const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 420_000 });
    const verifyIx = await coreBridge.processEncodedVaaIx(
      program,
      {
        writeAuthority: payer.publicKey,
        encodedVaa: encodedVaa.publicKey,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      },
      { verifySignaturesV1: {} }
    );

    await expectIxOk(
      program.provider.connection,
      [computeIx, createIx, initIx, firstProcessIx, verifyIx],
      [payer, encodedVaa]
    );
  }

  return encodedVaa.publicKey;
}
