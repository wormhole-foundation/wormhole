import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  GUARDIAN_KEYS,
  createAccountIx,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
  expectIxOkDetails,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { MockEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";

const GUARDIAN_SET_INDEX = 2;

const dummyEmitter = new MockEmitter(Buffer.alloc(32, "deadbeef").toString("hex"), 69, -1);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Process Encoded Vaa", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    const signedVaa = defaultVaa();
    const vaaSize = signedVaa.length;
    const chunkSize = 912; // Max that can fit in a transaction.

    it("Cannot Invoke `process_encoded_vaa` with Different Write Authority (Write)", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const someoneElse = anchor.web3.Keypair.generate();
      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: someoneElse.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { write: { index: 0, data: Buffer.from("Nope.") } }
      );
      await expectIxErr(connection, [ix], [payer, someoneElse], "WriteAuthorityMismatch");
    });

    it("Cannot Invoke `process_encoded_vaa` with Different Write Authority (CloseVaaAccount)", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const someoneElse = anchor.web3.Keypair.generate();
      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: someoneElse.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { closeVaaAccount: {} }
      );
      await expectIxErr(connection, [ix], [payer, someoneElse], "WriteAuthorityMismatch");
    });

    it("Cannot Invoke `process_encoded_vaa` with Expired Guardidan Set", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const expiredGuardianSetIndex = 0;
      expect(GUARDIAN_SET_INDEX).is.greaterThan(expiredGuardianSetIndex);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: coreBridge.GuardianSet.address(program.programId, expiredGuardianSetIndex),
        },
        { write: { index: 0, data: Buffer.from("Nope.") } }
      );
      await expectIxErr(connection, [ix], [payer], "GuardianSetExpired");
    });

    it("Cannot Invoke `process_encoded_vaa` with Nonsensical Index", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { write: { index: vaaSize, data: Buffer.from("Nope.") } }
      );
      await expectIxErr(connection, [ix], [payer], "DataOverflow");
    });

    it("Cannot Invoke `process_encoded_vaa` with Too Much Data", async () => {
      const smallVaaSize = 69;
      const encodedVaa = await initEncodedVaa(program, payer, smallVaaSize);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { write: { index: 0, data: Buffer.alloc(smallVaaSize + 1, "Nope.") } }
      );
      await expectIxErr(connection, [ix], [payer], "DataOverflow");
    });

    it("Cannot Invoke `process_encoded_vaa` with No Data", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { write: { index: 0, data: Buffer.alloc(0) } }
      );
      await expectIxErr(connection, [ix], [payer], "InvalidInstructionArgument");
    });

    it.skip("Cannot Invoke `process_encoded_vaa` to Verify Signatures on Nonsensical Encoded Vaa", async () => {
      // TODO
    });

    it("Invoke `process_encoded_vaa` to Close Encoded VAA", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const balanceBefore = await connection.getBalance(payer.publicKey);

      // const expectedLamports = await connection
      //   .getAccountInfo(encodedVaa)
      //   .then((acct) => acct.lamports);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { closeVaaAccount: {} }
      );
      //const txDetails = await expectIxOkDetails(connection, [ix], [payer]);
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

      it(`Invoke \`process_encoded_vaa\` to Write Part of VAA (Range: ${start}..${end})`, async () => {
        const encodedVaa = localVariables.get("encodedVaa") as anchor.web3.PublicKey;

        const ix = await coreBridge.processEncodedVaaIx(
          program,
          {
            writeAuthority: payer.publicKey,
            encodedVaa,
            guardianSet: null,
          },
          { write: { index: start, data: signedVaa.subarray(start, end) } }
        );
        await expectIxOk(connection, [ix], [payer]);

        const expectedBuf = Buffer.alloc(vaaSize);
        expectedBuf.set(signedVaa.subarray(0, end));

        const encodedVaaData = await coreBridge.EncodedVaa.fetch(program, encodedVaa);
        expectDeepEqual(encodedVaaData, {
          status: coreBridge.ProcessingStatus.Writing,
          writeAuthority: payer.publicKey,
          version: coreBridge.VaaVersion.Unset,
          buf: expectedBuf,
        });
      });
    }

    it("Cannot Invoke `process_encoded_vaa` to Verify Signatures without Guardian Set", async () => {
      const encodedVaa = localVariables.get("encodedVaa") as anchor.web3.PublicKey;

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { verifySignaturesV1: {} }
      );
      await expectIxErr(connection, [ix], [payer], "AccountNotEnoughKeys");
    });

    it("Invoke `process_encoded_vaa` to Verify Signatures", async () => {
      const encodedVaa = localVariables.get("encodedVaa") as anchor.web3.PublicKey;

      // This directive requires more than the usual 200k.
      const computeIx = anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 360_000 });

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
        },
        { verifySignaturesV1: {} }
      );
      await expectIxOk(connection, [computeIx, ix], [payer]);
    });

    it("Cannot Invoke `process_encoded_vaa` to Verify Signatures Again", async () => {
      const encodedVaa = localVariables.get("encodedVaa") as anchor.web3.PublicKey;

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX),
        },
        { verifySignaturesV1: {} }
      );
      await expectIxErr(connection, [ix], [payer], "VaaAlreadyVerified");
    });

    it("Invoke `process_encoded_vaa` to Close Encoded VAA After Verifying Signatures", async () => {
      const encodedVaa = await initEncodedVaa(program, payer, vaaSize);

      const ix = await coreBridge.processEncodedVaaIx(
        program,
        {
          writeAuthority: payer.publicKey,
          encodedVaa,
          guardianSet: null,
        },
        { closeVaaAccount: {} }
      );
      //const txDetails = await expectIxOkDetails(connection, [ix], [payer]);
      await expectIxOk(connection, [ix], [payer]);

      const encodedVaaData = await connection.getAccountInfo(encodedVaa);
      expect(encodedVaaData).is.null;
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

function defaultVaa(
  args?: {
    nonce?: number;
    payload?: Buffer;
    consistencyLevel?: number;
    timestamp?: number;
  },
  guardianIndices?: number[]
) {
  if (args === undefined) {
    args = {};
  }
  let { nonce, payload, consistencyLevel, timestamp } = args;

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