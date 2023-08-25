import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { execSync } from "child_process";
import * as fs from "fs";
import {
  GUARDIAN_KEYS,
  expectIxErr,
  expectIxOk,
  invokeVerifySignaturesAndPostVaa,
  loadProgramBpf,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";

const ARTIFACTS_PATH = `${__dirname}/../artifacts/wormhole_core_bridge_solana.so`;

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 2;
const GOVERNANCE_SEQUENCE = 1_098_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Legacy Instruction: Upgrade Contract", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

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
        coreBridge.upgradeAuthorityPda(program.programId)
      );

      localVariables.set("implementation", implementation);
      localVariables.set("cleanUpArtifacts", !exists);
    });

    it("Invoke `upgrade_contract` on Forked Core Bridge", async () => {
      const implementation = localVariables.get("implementation") as anchor.web3.PublicKey;

      // Create the signed VAA.
      const signedVaa = defaultVaa(implementation);

      await txOk(program, payer, signedVaa);

      // Save for later.
      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `upgrade_contract` with Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa") as Buffer;

      // Invoke the instruction.
      await expectIxErr(
        connection,
        [
          coreBridge.legacyUpgradeContractIx(
            program,
            { payer: payer.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "already in use"
      );
    });

    it("Deploy Same Implementation and Invoke `upgrade_contract` with Another VAA", async () => {
      const implementation = loadProgramBpf(
        ARTIFACTS_PATH,
        coreBridge.upgradeAuthorityPda(program.programId)
      );

      // Create the signed VAA.
      const signedVaa = defaultVaa(implementation);

      await txOk(program, payer, signedVaa);
    });
  });
});

function defaultVaa(implementation: anchor.web3.PublicKey, targetChain?: number): Buffer {
  const timestamp = 12345678;
  const published = governance.publishWormholeUpgradeContract(
    timestamp,
    targetChain === undefined ? 1 : targetChain,
    implementation.toString()
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 14]);
}

async function txOk(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  signedVaa: Buffer
) {
  const connection = program.provider.connection;

  // Parse the signed VAA.
  const parsedVaa = parseVaa(signedVaa);

  // Verify and Post.
  await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

  // Create the transferFees instruction.
  const ix = coreBridge.legacyUpgradeContractIx(program, { payer: payer.publicKey }, parsedVaa);

  // Invoke the instruction.
  return expectIxOk(connection, [ix], [payer]);
}
