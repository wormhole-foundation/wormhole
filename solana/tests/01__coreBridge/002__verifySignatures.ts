import { MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { ComputeBudgetProgram } from "@solana/web3.js";
import { expect } from "chai";
import { ethers } from "ethers";
import {
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  SignatureSets,
  createAccountIx,
  createSigVerifyIx,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

const GUARDIAN_SET_INDEX = 0;

const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Core Bridge -- Legacy Instruction: Verify Signatures", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    const accountConfigs: InvalidAccountConfig[] = [
      {
        label: "instructions",
        contextName: "instructions",
        errorMsg: "AccountSysvarMismatch",
      },
      {
        label: "guardian_set",
        contextName: "guardianSet",
        errorMsg: "ConstraintSeeds",
        dataLength: 4 + 4 + 20 * 19 + 4 + 4,
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const created = anchor.web3.Keypair.generate();

        if (cfg.dataLength !== undefined) {
          const ix = await createAccountIx(
            program.provider.connection,
            program.programId,
            payer,
            created,
            cfg.dataLength
          );
          await expectIxOk(connection, [ix], [payer, created]);
        }

        const signatureSet = anchor.web3.Keypair.generate();
        let accounts: coreBridge.LegacyVerifySignaturesContext = {
          payer: payer.publicKey,
          guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
          signatureSet: signatureSet.publicKey,
        };
        accounts[cfg.contextName] = created.publicKey;

        const ix = coreBridge.legacyVerifySignaturesIx(program, accounts, {
          signerIndices: new Array(19),
        });

        await expectIxErr(connection, [ix], [payer, signatureSet], cfg.errorMsg);
      });
    }

    it("Cannot Invoke `verify_signatures` without Preceding Instruction", async () => {
      const { signatureSet } = new SignatureSets();

      const signerIndices = new Array(19).fill(-1);
      signerIndices[3] = 0;

      const verifyIx = coreBridge.legacyVerifySignaturesIx(
        program,
        {
          payer: payer.publicKey,
          guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
          signatureSet: signatureSet.publicKey,
        },
        {
          signerIndices,
        }
      );
      await expectIxErr(connection, [verifyIx], [payer, signatureSet], "InstructionAtWrongIndex");
    });

    it("Cannot Invoke `verify_signatures` without Sig Verify Instruction", async () => {
      const computeIx = ComputeBudgetProgram.setComputeUnitLimit({ units: 696969 });

      const { signatureSet } = new SignatureSets();

      const signerIndices = new Array(19).fill(-1);
      signerIndices[3] = 0;

      const verifyIx = coreBridge.legacyVerifySignaturesIx(
        program,
        {
          payer: payer.publicKey,
          guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
          signatureSet: signatureSet.publicKey,
        },
        {
          signerIndices,
        }
      );
      await expectIxErr(
        connection,
        [computeIx, verifyIx],
        [payer, signatureSet],
        "InvalidSigVerifyInstruction"
      );
    });

    it("Cannot Invoke `verify_signatures` with Different Message on Existing Signature Set", async () => {
      const { signatureSet, forkSignatureSet } = new SignatureSets();
      const message = Buffer.from("I'm legitimate.");

      const guardianIndices = [1, 8, 9, 10, 11];
      await parallelIxOk(
        program,
        forkedProgram,
        payer,
        { signatureSet, forkSignatureSet },
        guardianIndices,
        message
      );

      const wrongMessage = Buffer.from("And I'm not.");
      const guardianIndex = 2;
      const sigVerifyIx = await createSigVerifyIx(
        program,
        guardians,
        GUARDIAN_SET_INDEX,
        wrongMessage,
        [guardianIndex]
      );

      const signerIndices = new Array(19).fill(-1);
      signerIndices[guardianIndex] = 0;

      const verifyIx = coreBridge.legacyVerifySignaturesIx(
        program,
        {
          payer: payer.publicKey,
          guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
          signatureSet: signatureSet.publicKey,
        },
        {
          signerIndices,
        }
      );
      await expectIxErr(
        connection,
        [sigVerifyIx, verifyIx],
        [payer, signatureSet],
        "MessageMismatch"
      );
    });

    it("Cannot Invoke `verify_signatures` with Signer Indices Mismatch", async () => {
      const { signatureSet } = new SignatureSets();
      const message = Buffer.from("Maybe legitimate.");
      const guardianIndices = [7, 8];
      const sigVerifyIx = await createSigVerifyIx(
        program,
        guardians,
        GUARDIAN_SET_INDEX,
        message,
        guardianIndices
      );

      // Only put one of the two indices in verify signatures ix.
      const signerIndices = new Array(19).fill(-1);
      signerIndices[guardianIndices[0]] = 0;

      const verifyIx = coreBridge.legacyVerifySignaturesIx(
        program,
        {
          payer: payer.publicKey,
          guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
          signatureSet: signatureSet.publicKey,
        },
        {
          signerIndices,
        }
      );
      await expectIxErr(
        connection,
        [sigVerifyIx, verifyIx],
        [payer, signatureSet],
        "SignerIndicesMismatch"
      );
    });

    it("Cannot Invoke `verify_signatures` with Invalid Guardian", async () => {
      const { signatureSet } = new SignatureSets();
      const message = Buffer.from("Maybe legitimate.");
      const guardianIndex = 7;
      const sigVerifyIx = await createSigVerifyIx(program, guardians, GUARDIAN_SET_INDEX, message, [
        guardianIndex,
      ]);

      const signerIndices = new Array(19).fill(-1);
      const wrongGuardianIndex = 8;
      expect(guardianIndex).not.equals(wrongGuardianIndex);
      signerIndices[wrongGuardianIndex] = 0;

      const verifyIx = coreBridge.legacyVerifySignaturesIx(
        program,
        {
          payer: payer.publicKey,
          guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
          signatureSet: signatureSet.publicKey,
        },
        {
          signerIndices,
        }
      );
      await expectIxErr(
        connection,
        [sigVerifyIx, verifyIx],
        [payer, signatureSet],
        "InvalidGuardianKeyRecovery"
      );
    });
  });

  describe("Ok", () => {
    const message = Buffer.from("Not a Wormhole message.");

    // This signature set will be written multiple times.
    const { signatureSet, forkSignatureSet } = new SignatureSets();

    for (let i = 0; i < 17; i += 2) {
      it(`Invoke \`verify_signatures\` for Overlapping Guardians [${i}], [${i + 1}], [${
        i + 2
      }]`, async () => {
        await parallelIxOk(
          program,
          forkedProgram,
          payer,
          { signatureSet, forkSignatureSet },
          [i, i + 1, i + 2],
          message
        );

        const [signatureSetData, forkSignatureSetData] = await Promise.all([
          coreBridge.SignatureSet.fromAccountAddress(connection, signatureSet.publicKey),
          coreBridge.SignatureSet.fromAccountAddress(connection, forkSignatureSet.publicKey),
        ]);
        const sigVerifySuccesses: boolean[] = new Array(19).fill(false);
        for (let j = 0; j <= i + 2; ++j) {
          sigVerifySuccesses[j] = true;
        }
        expectDeepEqual(signatureSetData, {
          sigVerifySuccesses,
          messageHash: Array.from(ethers.utils.arrayify(ethers.utils.keccak256(message))),
          guardianSetIndex: GUARDIAN_SET_INDEX,
        });
        expectDeepEqual(signatureSetData, forkSignatureSetData);
      });
    }
  });
});

async function parallelIxOk(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  signatureSets: SignatureSets,
  guardianIndices: number[],
  message: Buffer
) {
  const sigVerifyIx = await createSigVerifyIx(
    program,
    guardians,
    GUARDIAN_SET_INDEX,
    message,
    guardianIndices
  );
  const signerIndices = new Array(19).fill(-1);
  let count = 0;
  for (const i of guardianIndices) {
    signerIndices[i] = count;
    ++count;
  }

  const { signatureSet, forkSignatureSet } = signatureSets;

  const guardianSet = coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX);
  const verifyIx = coreBridge.legacyVerifySignaturesIx(
    program,
    { payer: payer.publicKey, guardianSet, signatureSet: signatureSet.publicKey },
    {
      signerIndices,
    }
  );

  const forkGuardianSet = coreBridge.GuardianSet.address(
    forkedProgram.programId,
    GUARDIAN_SET_INDEX
  );
  const forkVerifyIx = coreBridge.legacyVerifySignaturesIx(
    forkedProgram,
    {
      payer: payer.publicKey,
      guardianSet: forkGuardianSet,
      signatureSet: forkSignatureSet.publicKey,
    },
    {
      signerIndices,
    }
  );

  const connection = program.provider.connection;
  await Promise.all([
    expectIxOk(connection, [sigVerifyIx, verifyIx], [payer, signatureSet]),
    expectIxOk(connection, [sigVerifyIx, forkVerifyIx], [payer, forkSignatureSet]),
  ]);
}
