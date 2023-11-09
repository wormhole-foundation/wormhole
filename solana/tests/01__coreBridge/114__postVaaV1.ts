import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  GUARDIAN_KEYS,
  createAccountIx,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
  expectIxOkDetails,
  processVaa,
  transferLamports,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { MockEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import { parseVaa } from "@certusone/wormhole-sdk";

const GUARDIAN_SET_INDEX = 4;

const dummyEmitter = new MockEmitter(Buffer.alloc(32, "deadbeef").toString("hex"), 69, -1);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Post VAA V1", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    it("Cannot Invoke `post_vaa_v1` with Incorrect Post VAA PDA", async () => {
      const signedVaa = defaultVaa();

      const encodedVaa = await processVaa(program, payer, signedVaa, GUARDIAN_SET_INDEX);

      const ix = await coreBridge.postVaaV1Ix(program, {
        payer: payer.publicKey,
        encodedVaa,
        postedVaa: coreBridge.PostedVaaV1.address(program.programId, new Array(32)),
      });
      await expectIxErr(connection, [ix], [payer], "ConstraintSeeds");
    });

    it("Cannot Invoke `post_vaa_v1` with Unverified VAA", async () => {
      const signedVaa = defaultVaa();

      const encodedVaa = await processVaa(
        program,
        payer,
        signedVaa,
        GUARDIAN_SET_INDEX,
        false // verify
      );
      const vaaData = await coreBridge.EncodedVaa.fetch(program, encodedVaa);
      expect(vaaData.status).not.equals(coreBridge.ProcessingStatus.Verified);

      const ix = await coreBridge.postVaaV1Ix(program, {
        payer: payer.publicKey,
        encodedVaa,
      });
      await expectIxErr(connection, [ix], [payer], "UnverifiedVaa");
    });

    it("Invoke `post_vaa_v1`", async () => {
      const signedVaa = defaultVaa();

      const encodedVaa = await processVaa(program, payer, signedVaa, GUARDIAN_SET_INDEX);

      const ix = await coreBridge.postVaaV1Ix(program, {
        payer: payer.publicKey,
        encodedVaa,
      });
      await expectIxOk(connection, [ix], [payer]);

      const postedVaaData = await coreBridge.PostedVaaV1.fromPda(
        connection,
        program.programId,
        Array.from(parseVaa(signedVaa).hash)
      );
      expectDeepEqual(postedVaaData, {
        consistencyLevel: 200,
        timestamp: 12345678,
        signatureSet: anchor.web3.PublicKey.default,
        guardianSetIndex: GUARDIAN_SET_INDEX,
        nonce: 420,
        sequence: new anchor.BN(2),
        emitterChain: 69,
        emitterAddress: Array.from(Buffer.alloc(32, "deadbeef")),
        payload: Buffer.alloc(2 * 1_024, "Somebody set us up the bomb. "),
      });

      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `post_vaa_v1` with Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa") as Buffer;

      const encodedVaa = await processVaa(program, payer, signedVaa, GUARDIAN_SET_INDEX);

      const ix = await coreBridge.postVaaV1Ix(program, {
        payer: payer.publicKey,
        encodedVaa,
      });
      await expectIxErr(connection, [ix], [payer], "already in use");
    });
  });
});

function defaultVaa(
  args: {
    vaaLen?: number;
    nonce?: number;
    payload?: Buffer;
    consistencyLevel?: number;
    timestamp?: number;
  } = {},
  guardianIndices?: number[]
) {
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
