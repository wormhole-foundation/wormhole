import { ParsedVaa, parseVaa } from "@certusone/wormhole-sdk";
import { MockEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import { ethers } from "ethers";
import {
  GUARDIAN_KEYS,
  SignatureSets,
  expectDeepEqual,
  expectIxErr,
  expectIxOkDetails,
  parallelVerifySignatures,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

const GUARDIAN_SET_INDEX = 0;

const dummyEmitter = new MockEmitter(Buffer.alloc(32, "deadbeef").toString("hex"), 69, -1);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Core Bridge -- Legacy Instruction: Post VAA", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  // Test variables.
  const localVariables = new Map<string, any>();

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    it("Invoke `post_vaa`", async () => {
      const signedVaa = defaultVaa();

      const { signatureSet, forkSignatureSet, args, parsed, messageHash } = await createArgs(
        connection,
        payer,
        signedVaa
      );
      expectDeepEqual(parsed.hash, Buffer.from(messageHash));

      await parallelIxOk(program, forkedProgram, { payer: payer.publicKey }, args, payer, {
        signatureSet,
        forkSignatureSet,
      });

      const [postedVaaData, forkPostedVaaData] = await Promise.all([
        coreBridge.PostedVaaV1.fromPda(connection, program.programId, messageHash),
        coreBridge.PostedVaaV1.fromPda(connection, forkedProgram.programId, messageHash),
      ]);

      // Signature set accounts are different, so we cannot do a deep equal compare. But we'll be
      // close enough by checking each field.
      const fields = ["consistencyLevel", "timestamp", "nonce", "emitterChain"];
      for (const field of fields) {
        expect(postedVaaData[field]).to.equal(forkPostedVaaData[field]);
      }
      const deepFields = ["sequence", "emitterAddress", "payload"];
      for (const field of deepFields) {
        expectDeepEqual(postedVaaData[field], forkPostedVaaData[field]);
      }
      expectDeepEqual(postedVaaData.signatureSet, signatureSet.publicKey);
      expectDeepEqual(forkPostedVaaData.signatureSet, forkSignatureSet.publicKey);

      // Patched the Posted VAA account to save guardian set index now. Legacy program does not
      // save this field.
      expect(postedVaaData.guardianSetIndex).to.equal(GUARDIAN_SET_INDEX);
      expect(forkPostedVaaData.guardianSetIndex).to.equal(0);

      // Now compare parsed VAA fields.
      expect(postedVaaData.consistencyLevel).to.equal(parsed.consistencyLevel);
      expect(postedVaaData.timestamp).to.equal(parsed.timestamp);
      expect(postedVaaData.nonce).to.equal(parsed.nonce);
      expect(postedVaaData.emitterChain).to.equal(parsed.emitterChain);
      expect(postedVaaData.sequence.toString()).to.equal(parsed.sequence.toString());
      expectDeepEqual(postedVaaData.emitterAddress, Array.from(parsed.emitterAddress));
      expectDeepEqual(postedVaaData.payload, parsed.payload);

      // Save Vaa to local variables.
      localVariables.set("signedVaa", signedVaa);
    });
  });

  describe("New Implementation", () => {
    it("Cannot Invoke `post_vaa` With Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa") as Buffer;

      const { signatureSet, args } = await createArgs(connection, payer, signedVaa);

      const ix = coreBridge.legacyPostVaaIx(
        program,
        { signatureSet: signatureSet.publicKey, payer: payer.publicKey },
        args
      );

      await expectIxErr(connection, [ix], [payer], "already in use");
    });

    it("Cannot Invoke `post_vaa` No Quorum", async () => {
      // Specify less than 13 guardians, so quorum is not reached.
      const signedVaa = defaultVaa({ guardianIndices: [0, 1, 2, 3, 4, 5] });

      const { signatureSet, args, parsed, messageHash } = await createArgs(
        connection,
        payer,
        signedVaa
      );
      expectDeepEqual(parsed.hash, Buffer.from(messageHash));

      const ix = coreBridge.legacyPostVaaIx(
        program,
        { signatureSet: signatureSet.publicKey, payer: payer.publicKey },
        args
      );

      await expectIxErr(connection, [ix], [payer], "NoQuorum");
    });

    it("Cannot Invoke `post_vaa` Invalid Message Hash", async () => {
      const signedVaa = defaultVaa({ nonce: 69 });
      const { signatureSet, args, messageHash } = await createArgs(connection, payer, signedVaa);

      // Fetch the expected posted_vaa account.
      const postedVaa = await coreBridge.PostedVaaV1.address(program.programId, messageHash);

      // Update the arguments that are passed to the instruction, so that the message hash
      // is incorrect when it's recomputed in the program.
      args.nonce = 42069;

      const ix = coreBridge.legacyPostVaaIx(
        program,
        { signatureSet: signatureSet.publicKey, payer: payer.publicKey, postedVaa },
        args
      );

      await expectIxErr(connection, [ix], [payer], "InvalidMessageHash");
    });
  });
});

type DefaultArgsOutput = {
  signatureSet: anchor.web3.Keypair;
  forkSignatureSet: anchor.web3.Keypair;
  args: coreBridge.LegacyPostVaaArgs;
  parsed: ParsedVaa;
  messageHash: number[];
};

function computeMessageHash(args: coreBridge.LegacyPostVaaArgs): number[] {
  const { timestamp, nonce, emitterChain, emitterAddress, sequence, consistencyLevel, payload } =
    args;
  const message = Buffer.alloc(51 + payload.length);
  message.writeUInt32BE(timestamp, 0);
  message.writeUInt32BE(nonce, 4);
  message.writeUInt16BE(emitterChain, 8);
  message.set(emitterAddress, 10);
  message.writeBigUInt64BE(BigInt(sequence.toString()), 42);
  message.writeUInt8(consistencyLevel, 50);
  message.set(payload, 51);

  return Array.from(ethers.utils.arrayify(ethers.utils.keccak256(message)));
}

async function createArgs(
  connection: anchor.web3.Connection,
  payer: anchor.web3.Keypair,
  signedVaa: Buffer
): Promise<DefaultArgsOutput> {
  const { signatureSet, forkSignatureSet } = await parallelVerifySignatures(
    connection,
    payer,
    signedVaa
  );

  const parsed = parseVaa(signedVaa);
  const args = {
    version: parsed.version,
    guardianSetIndex: parsed.guardianSetIndex,
    timestamp: parsed.timestamp,
    nonce: parsed.nonce,
    emitterChain: parsed.emitterChain,
    emitterAddress: Array.from(parsed.emitterAddress),
    sequence: new anchor.BN(parsed.sequence.toString()),
    consistencyLevel: parsed.consistencyLevel,
    payload: parsed.payload,
  };

  return {
    signatureSet,
    forkSignatureSet,
    args,
    parsed,
    messageHash: computeMessageHash(args),
  };
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

  if (args.nonce === undefined) {
    args.nonce = 420;
  }

  if (args.payload === undefined) {
    args.payload = Buffer.from("All your base are belong to us.");
  }

  if (args.consistencyLevel === undefined) {
    args.consistencyLevel = 200;
  }

  if (args.timestamp === undefined) {
    args.timestamp = 12345678;
  }

  if (args.guardianIndices === undefined) {
    args.guardianIndices = [0, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 14];
  }

  const published = dummyEmitter.publishMessage(
    args.nonce,
    args.payload,
    args.consistencyLevel,
    args.timestamp
  );
  return guardians.addSignatures(published, args.guardianIndices);
}

async function parallelIxOk(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: { payer: anchor.web3.PublicKey },
  args: coreBridge.LegacyPostVaaArgs,
  payer: anchor.web3.Keypair,
  signatureSets: SignatureSets
) {
  const connection = program.provider.connection;
  const { signatureSet, forkSignatureSet } = signatureSets;
  const ix = coreBridge.legacyPostVaaIx(
    program,
    { signatureSet: signatureSet.publicKey, ...accounts },
    args
  );
  const forkedIx = coreBridge.legacyPostVaaIx(
    forkedProgram,
    { signatureSet: forkSignatureSet.publicKey, ...accounts },
    args
  );
  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer]),
    expectIxOkDetails(connection, [forkedIx], [payer]),
  ]);
}
