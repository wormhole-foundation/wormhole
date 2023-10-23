import {
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  parseVaa,
  tryNativeToHexString,
} from "@certusone/wormhole-sdk";
import {
  GovernanceEmitter,
  MockEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { execSync } from "child_process";
import * as fs from "fs";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  TOKEN_BRIDGE_GOVERNANCE_MODULE,
  expectIxErr,
  expectIxOk,
  invokeVerifySignaturesAndPostVaa,
  loadProgramBpf,
} from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";

const ARTIFACTS_PATH = `${__dirname}/../artifacts/wormhole_token_bridge_solana.so`;

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 4;
const GOVERNANCE_SEQUENCE = 2_098_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Token Bridge -- Legacy Instruction: Upgrade Contract", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  after("Clean Up", async () => {
    const cleanUp = localVariables.get("cleanUpArtifacts") as boolean;
    if (cleanUp) {
      fs.rmSync(ARTIFACTS_PATH, { force: true, recursive: true });
    }
  });

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    it("Deploy Implementation", async () => {
      const exists = fs.existsSync(ARTIFACTS_PATH);
      if (!exists) {
        // Need to build.
        execSync(`cd ${__dirname}/../.. && NETWORK=mainnet make build`);
      }

      const implementation = loadProgramBpf(
        ARTIFACTS_PATH,
        tokenBridge.upgradeAuthorityPda(program.programId)
      );

      localVariables.set("implementation", implementation);
      localVariables.set("cleanUpArtifacts", !exists);
    });

    it("Invoke `upgrade_contract` on Forked Core Bridge", async () => {
      const implementation = localVariables.get("implementation") as anchor.web3.PublicKey;

      // Create the signed VAA.
      const signedVaa = defaultVaa(implementation);

      await sendTx(program, payer, signedVaa);

      // Save for later.
      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `upgrade_contract` with Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa") as Buffer;

      // Invoke the instruction.
      await expectIxErr(
        connection,
        [
          tokenBridge.legacyUpgradeContractIx(
            program,
            { payer: payer.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "invalid account data for instruction"
      );
    });

    it("Deploy Same Implementation and Invoke `upgrade_contract` with Another VAA", async () => {
      const implementation = loadProgramBpf(
        ARTIFACTS_PATH,
        tokenBridge.upgradeAuthorityPda(program.programId)
      );

      // Create the signed VAA.
      const signedVaa = defaultVaa(implementation);

      await sendTx(program, payer, signedVaa);
    });

    it("Cannot Invoke `upgrade_contract` with Implementation Mismatch", async () => {
      const realImplementation = loadProgramBpf(
        ARTIFACTS_PATH,
        tokenBridge.upgradeAuthorityPda(program.programId)
      );

      // Create the signed VAA with a random implementation.
      const signedVaa = defaultVaa(anchor.web3.Keypair.generate().publicKey);

      // Verify and Post.
      await invokeVerifySignaturesAndPostVaa(
        tokenBridge.getCoreBridgeProgram(program),
        payer,
        signedVaa
      );

      // Create the upgrade instruction, but pass a buffer with the realImplementation pubkey.
      const ix = tokenBridge.legacyUpgradeContractIx(
        program,
        { payer: payer.publicKey, buffer: realImplementation },
        parseVaa(signedVaa)
      );

      await expectIxErr(connection, [ix], [payer], "ImplementationMismatch");
    });

    it("Cannot Invoke `register_chain` with Invalid Emitter Address", async () => {
      const invalidGovernanceEmitter = tryNativeToHexString(
        ETHEREUM_TOKEN_BRIDGE_ADDRESS,
        CHAIN_ID_ETH
      );
      const governanceChain = CHAIN_ID_SOLANA;
      const sequence = 1;
      const implementation = loadProgramBpf(
        ARTIFACTS_PATH,
        tokenBridge.upgradeAuthorityPda(program.programId)
      );

      // Create a bogus governance VAA.
      const signedVaa = createInvalidUpgradeVaa(
        invalidGovernanceEmitter,
        governanceChain,
        sequence,
        implementation.toBuffer()
      );

      await sendTx(program, payer, signedVaa, "InvalidGovernanceEmitter");
    });

    it("Cannot Invoke `register_chain` with Invalid Emitter Chain", async () => {
      const invalidGovernanceChain = CHAIN_ID_ETH;
      const sequence = 2;
      const implementation = loadProgramBpf(
        ARTIFACTS_PATH,
        tokenBridge.upgradeAuthorityPda(program.programId)
      );

      // Create a bogus governance VAA.
      const signedVaa = createInvalidUpgradeVaa(
        GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
        invalidGovernanceChain,
        sequence,
        implementation.toBuffer()
      );

      await sendTx(program, payer, signedVaa, "InvalidGovernanceEmitter");
    });

    it("Cannot Invoke `register_chain` with Invalid Governance Module", async () => {
      const governanceChain = CHAIN_ID_SOLANA;
      const sequence = 3;
      const invalidGovernanceModule = Buffer.from(
        "00000000000000000000000000000000000000000000000000000000deadbeef",
        "hex"
      );
      const implementation = loadProgramBpf(
        ARTIFACTS_PATH,
        tokenBridge.upgradeAuthorityPda(program.programId)
      );

      // Create a bogus governance VAA.
      const signedVaa = createInvalidUpgradeVaa(
        GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
        governanceChain,
        sequence,
        implementation.toBuffer(),
        invalidGovernanceModule
      );

      await sendTx(program, payer, signedVaa, "InvalidGovernanceVaa");
    });

    it("Cannot Invoke `register_chain` with Invalid Governance Action", async () => {
      const governanceChain = CHAIN_ID_SOLANA;
      const sequence = 4;
      const invalidGovernanceAction = 69;
      const implementation = loadProgramBpf(
        ARTIFACTS_PATH,
        tokenBridge.upgradeAuthorityPda(program.programId)
      );

      // Create a bogus governance VAA.
      const signedVaa = createInvalidUpgradeVaa(
        GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
        governanceChain,
        sequence,
        implementation.toBuffer(),
        Buffer.from(TOKEN_BRIDGE_GOVERNANCE_MODULE, "hex"),
        invalidGovernanceAction
      );

      await sendTx(program, payer, signedVaa, "InvalidGovernanceVaa");
    });
  });
});

function defaultVaa(implementation: anchor.web3.PublicKey, targetChain?: number): Buffer {
  const timestamp = 12345678;
  const published = governance.publishTokenBridgeUpgradeContract(
    timestamp,
    targetChain === undefined ? 1 : targetChain,
    implementation.toString()
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 14]);
}

function createInvalidUpgradeVaa(
  emitter: string,
  chain: number,
  sequence: number,
  implementation: Buffer,
  governanceModule?: Buffer,
  governanceAction?: number
): Buffer {
  const mockEmitter = new MockEmitter(emitter, chain, sequence);

  if (governanceModule === undefined) {
    governanceModule = Buffer.from(
      "000000000000000000000000000000000000000000546f6b656e427269646765",
      "hex"
    );
  }

  if (governanceAction === undefined) {
    governanceAction = 2;
  }

  // Mock register chain payload.
  let payload = Buffer.alloc(69);
  payload.set(governanceModule, 0);
  payload.writeUint8(governanceAction, 32);
  payload.writeUint16BE(0, 33);
  payload.set(implementation, 35);

  // Vaa info.
  const published = mockEmitter.publishMessage(
    69, // Nonce.
    payload,
    1 // Finality.
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function sendTx(
  program: tokenBridge.TokenBridgeProgram,
  payer: anchor.web3.Keypair,
  signedVaa: Buffer,
  expectedError?: string
) {
  const connection = program.provider.connection;

  // Parse the signed VAA.
  const parsedVaa = parseVaa(signedVaa);

  // Verify and Post.
  await invokeVerifySignaturesAndPostVaa(
    tokenBridge.getCoreBridgeProgram(program),
    payer,
    signedVaa
  );

  // Create the transferFees instruction.
  const ix = tokenBridge.legacyUpgradeContractIx(program, { payer: payer.publicKey }, parsedVaa);

  // Invoke the instruction.
  if (expectedError === undefined) {
    return expectIxOk(connection, [ix], [payer]);
  } else {
    return expectIxErr(connection, [ix], [payer], expectedError);
  }
}
