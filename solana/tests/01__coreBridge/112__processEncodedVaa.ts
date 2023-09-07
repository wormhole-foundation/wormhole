import {
  GovernanceEmitter,
  MockEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { ComputeBudgetProgram } from "@solana/web3.js";
import { expect } from "chai";
import {
  GUARDIAN_KEYS,
  createAccountIx,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
  processVaa,
  parallelPostVaa,
  GOVERNANCE_EMITTER_ADDRESS,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { parseVaa } from "@certusone/wormhole-sdk";

const GUARDIAN_SET_INDEX = 2;

const dummyEmitter = new MockEmitter(Buffer.alloc(32, "deadbeef").toString("hex"), 69, -1);
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  1_015_000 - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Process Encoded Vaa", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  // We need to use the localnet program to keep the Wormhole programs in sync. This
  // test suite updates the guardian set.
  const localnetProgram = coreBridge.getAnchorProgram(connection, coreBridge.localnet());

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    const signedVaa = defaultVaa();
    const vaaSize = signedVaa.length;
    const chunkSize = 912; // Max that can fit in a transaction.

    it("Cannot Invoke `write_encoded_vaa` with Different Write Authority", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const someoneElse = anchor.web3.Keypair.generate();
      const ix = await coreBridge.writeEncodedVaaIx(
        program,
        {
          writeAuthority: someoneElse.publicKey,
          encodedVaa,
        },
        { index: 0, data: Buffer.from("Nope.") }
      );
      await expectIxErr(connection, [ix], [payer, someoneElse], "WriteAuthorityMismatch");
    });

    it("Cannot Invoke `close_encoded_vaa` with Different Write Authority", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const someoneElse = anchor.web3.Keypair.generate();
      const ix = await coreBridge.closeEncodedVaaIx(program, {
        writeAuthority: someoneElse.publicKey,
        encodedVaa,
      });
      await expectIxErr(connection, [ix], [payer, someoneElse], "WriteAuthorityMismatch");
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` with Expired Guardian Set", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const expiredGuardianSetIndex = 0;
      expect(GUARDIAN_SET_INDEX).is.greaterThan(expiredGuardianSetIndex);

      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, expiredGuardianSetIndex),
      });
      await expectIxErr(connection, [ix], [payer], "GuardianSetExpired");
    });

    it("Cannot Invoke `write_encoded_vaa` with Nonsensical Index", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const ix = await coreBridge.writeEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
        },
        { index: vaaSize, data: Buffer.from("Nope.") }
      );
      await expectIxErr(connection, [ix], [payer], "DataOverflow");
    });

    it("Cannot Invoke `write_encoded_vaa` with Too Much Data", async () => {
      const smallVaaSize = 69;
      const encodedVaa = await initEncodedVaa(program, payer, smallVaaSize);

      const ix = await coreBridge.writeEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
        },
        { index: 0, data: Buffer.alloc(smallVaaSize + 1, "Nope.") }
      );
      await expectIxErr(connection, [ix], [payer], "DataOverflow");
    });

    it("Cannot Invoke `write_encoded_vaa` with No Data", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const ix = await coreBridge.writeEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
        },
        { index: 0, data: Buffer.alloc(0) }
      );
      await expectIxErr(connection, [ix], [payer], "InvalidInstructionArgument");
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` with Invalid VAA Version", async () => {
      let signedVaa = defaultVaa();

      // Spoof the VAA version number.
      signedVaa.writeUInt8(0x69, 0);

      const encodedVaa = await processVaa(
        program,
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX,
        false // verify
      );

      // This directive requires more than the usual 200k.
      const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });

      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      });
      await expectIxErr(connection, [computeIx, ix], [payer], "InvalidVaaVersion");
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` with Non-Increasing Guardian Index", async () => {
      let signedVaa = defaultVaa();

      // Change the second guardian index to the same value as the first.
      signedVaa.writeUInt8(0x0, 72);

      const encodedVaa = await processVaa(
        program,
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX,
        false // verify
      );

      // This directive requires more than the usual 200k.
      const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });

      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      });
      await expectIxErr(connection, [computeIx, ix], [payer], "InvalidGuardianIndex");
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` with Non-Existent Guardian Index", async () => {
      let signedVaa = defaultVaa();

      // Change the first guardian index to a value that doesn't exist in the
      // guardian set.
      signedVaa.writeUInt8(0x69, 6);

      const encodedVaa = await processVaa(
        program,
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX,
        false // verify
      );

      // This directive requires more than the usual 200k.
      const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });

      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      });
      await expectIxErr(connection, [computeIx, ix], [payer], "InvalidGuardianIndex");
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` with Invalid Guardian Key Recovery", async () => {
      let signedVaa = defaultVaa();

      // Change the recovery key of the first signature to zero, this will cause the key
      // recovery to fail.
      signedVaa.writeUInt8(0x0, 71);

      const encodedVaa = await processVaa(
        program,
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX,
        false // verify
      );

      // This directive requires more than the usual 200k.
      const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });

      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      });
      await expectIxErr(connection, [computeIx, ix], [payer], "InvalidGuardianKeyRecovery");
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` with Invalid Signature", async () => {
      let signedVaa = defaultVaa();

      // Change the recovery key of the first signature to an invalid value,
      // this will cause the signature verification to fail.
      signedVaa.writeUInt8(0x69, 71);

      const encodedVaa = await processVaa(
        program,
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX,
        false // verify
      );

      // This directive requires more than the usual 200k.
      const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });

      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      });
      await expectIxErr(connection, [computeIx, ix], [payer], "InvalidSignature");
    });

    it.skip("Cannot Invoke `verify_encoded_vaa_v1` to Verify Signatures on Nonsensical Encoded Vaa", async () => {
      // TODO
    });

    it("Invoke `close_encoded_vaa` to Close Encoded VAA", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const balanceBefore = await connection.getBalance(payer.publicKey);

      const ix = await coreBridge.closeEncodedVaaIx(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
      });
      await expectIxOk(connection, [ix], [payer]);

      const balanceAfter = await connection.getBalance(payer.publicKey);

      // Cannot reconcile expected lamports with lamport change on payer. But we show that the
      // balance increases, so the closed account lamports must have been sent to the payer.
      expect(balanceAfter).is.greaterThan(balanceBefore);

      const encodedVaaData = await connection.getAccountInfo(encodedVaa);
      expect(encodedVaaData).is.null;
    });

    it(`Invoke \`init_encoded_vaa\` on VAA Size == ${vaaSize}`, async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      localVariables.set("encodedVaa", encodedVaa);
    });

    for (let start = 0; start < vaaSize; start += chunkSize) {
      const end = Math.min(start + chunkSize, vaaSize);

      it(`Invoke \`write_encoded_vaa\` to Write Part of VAA (Range: ${start}..${end})`, async () => {
        const encodedVaa = localVariables.get("encodedVaa") as anchor.web3.PublicKey;

        const ix = await coreBridge.writeEncodedVaaIx(
          program,
          {
            writeAuthority: payer.publicKey,
            encodedVaa,
          },
          { index: start, data: signedVaa.subarray(start, end) }
        );
        await expectIxOk(connection, [ix], [payer]);

        const expectedBuf = Buffer.alloc(vaaSize);
        expectedBuf.set(signedVaa.subarray(0, end));

        const encodedVaaData = await coreBridge.EncodedVaa.fetch(program, encodedVaa);
        expectDeepEqual(encodedVaaData, {
          status: coreBridge.ProcessingStatus.Writing,
          writeAuthority: payer.publicKey,
          version: 0,
          buf: expectedBuf,
        });
      });
    }

    it("Invoke `verify_encoded_vaa_v1` to Verify Signatures", async () => {
      const encodedVaa = localVariables.get("encodedVaa") as anchor.web3.PublicKey;

      // This directive requires more than the usual 200k.
      const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });

      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      });
      await expectIxOk(connection, [computeIx, ix], [payer]);
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` to Verify Signatures Again", async () => {
      const encodedVaa = localVariables.get("encodedVaa") as anchor.web3.PublicKey;

      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      });
      await expectIxErr(connection, [ix], [payer], "VaaAlreadyVerified");
    });

    it("Invoke `close_encoded_vaa` to Close Encoded VAA After Verifying Signatures", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const ix = await coreBridge.closeEncodedVaaIx(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
      });
      //const txDetails = await expectIxOkDetails(connection, [ix], [payer]);
      await expectIxOk(connection, [ix], [payer]);

      const encodedVaaData = await connection.getAccountInfo(encodedVaa);
      expect(encodedVaaData).is.null;
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` to Verify Signatures without Quorum", async () => {
      const guardianIndices = [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12];
      expect(guardianIndices).has.length.lessThan((2 * guardians.signers.length) / 3);

      const badVaa = defaultVaa({ payload: Buffer.from("Unverified."), guardianIndices });
      const encodedVaa = await initEncodedVaa(program, payer, badVaa.length);

      const computeIx = ComputeBudgetProgram.setComputeUnitLimit({ units: 400_000 });
      const writeIx = await coreBridge.writeEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
        },
        { index: 0, data: badVaa }
      );

      const verifyIx = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
      });

      await expectIxErr(connection, [computeIx, writeIx, verifyIx], [payer], "NoQuorum");
    });

    it("Cannot Invoke `verify_encoded_vaa_v1` to Verify Signatures with Mismatching Guardian Set", async () => {
      // Sign a VAA with the current guardian set.
      const encodedVaa = await processVaa(
        program,
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX,
        false // verify
      );

      // Save the current guardian set.
      const currentGuardianSet = guardians.getPublicKeys();
      const newGuardianSet = guardians.setIndex + 1;

      // Update the guardian set.
      await updateGuardianSet(
        program,
        localnetProgram,
        payer,
        newGuardianSet,
        currentGuardianSet.slice(0, 3),
        [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14]
      );
      guardians.updateGuardianSetIndex(newGuardianSet);

      // This directive requires more than the usual 200k.
      const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });

      // Create the instruction using the newest guardian set.
      const ix = await coreBridge.verifyEncodedVaaV1Ix(program, {
        writeAuthority: payer.publicKey,
        encodedVaa,
        guardianSet: coreBridge.GuardianSet.address(program.programId, newGuardianSet),
      });
      await expectIxErr(connection, [computeIx, ix], [payer], "GuardianSetMismatch");

      // Revert the guardian set by updating to the original set.
      await updateGuardianSet(
        program,
        localnetProgram,
        payer,
        newGuardianSet + 1,
        currentGuardianSet,
        [0, 1, 2]
      );
    });
  });
});

async function initEncodedVaa(
  program: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  vaaSize: number
) {
  const encodedVaa = anchor.web3.Keypair.generate();
  const createIx = await createAccountIx(
    program.provider.connection,
    program.programId,
    payer,
    encodedVaa,
    46 + vaaSize
  );

  const initIx = await coreBridge.initEncodedVaaIx(program, {
    writeAuthority: payer.publicKey,
    encodedVaa: encodedVaa.publicKey,
  });

  await expectIxOk(program.provider.connection, [createIx, initIx], [payer, encodedVaa]);

  return encodedVaa.publicKey;
}

function defaultVaa(args?: {
  nonce?: number;
  payload?: Buffer;
  consistencyLevel?: number;
  timestamp?: number;
  guardianIndices?: number[];
}) {
  if (args === undefined) {
    args = {};
  }
  let { nonce, payload, consistencyLevel, timestamp, guardianIndices } = args;

  if (nonce === undefined) {
    nonce = 420;
  }

  if (payload === undefined) {
    payload = Buffer.alloc(2 * 1_024, "Somebody set us up the bomb. ");
  }

  if (consistencyLevel === undefined) {
    consistencyLevel = 200;
  }

  if (timestamp === undefined) {
    timestamp = 12345678;
  }

  if (guardianIndices === undefined) {
    guardianIndices = [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14];
  }

  const published = dummyEmitter.publishMessage(nonce, payload, consistencyLevel, timestamp);

  return guardians.addSignatures(published, guardianIndices);
}

async function updateGuardianSet(
  program: coreBridge.CoreBridgeProgram,
  localProgram: coreBridge.CoreBridgeProgram,
  payer: anchor.web3.Keypair,
  newIndex: number,
  newKeys: Buffer[],
  keyRange: number[]
) {
  const timestamp = 294967295;
  const published = governance.publishWormholeGuardianSetUpgrade(timestamp, newIndex, newKeys);
  const signedVaa = guardians.addSignatures(published, keyRange);

  // Parse the signed VAA.
  const parsedVaa = parseVaa(signedVaa);

  // Verify and Post
  await parallelPostVaa(program.provider.connection, payer, signedVaa);

  // Update the guardian set for both the upgraded program and the localnet program.
  const ix = coreBridge.legacyGuardianSetUpdateIx(program, { payer: payer.publicKey }, parsedVaa);
  const localIx = coreBridge.legacyGuardianSetUpdateIx(
    localProgram,
    { payer: payer.publicKey },
    parsedVaa
  );

  await expectIxOk(program.provider.connection, [ix, localIx], [payer]);
}
