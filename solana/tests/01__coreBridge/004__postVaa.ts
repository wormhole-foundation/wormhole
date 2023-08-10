import * as anchor from "@coral-xyz/anchor";
import { ethers } from "ethers";
import {
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  InvalidArgConfig,
  expectDeepEqual,
  expectIxErr,
  expectIxOkDetails,
  invokeVerifySignatures,
  parallelVerifySignatures,
  sleep,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";
import { MockEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import { parseVaa } from "@certusone/wormhole-sdk";

const GUARDIAN_SET_INDEX = 0;

const dummyEmitter = new MockEmitter(Buffer.alloc(32, "deadbeef").toString("hex"), 69, -1);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Core Bridge -- Legacy Instruction: Post VAA", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(
    connection,
    coreBridge.getProgramId("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
  );
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = coreBridge.getAnchorProgram(
    connection,
    coreBridge.getProgramId("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth")
  );

  describe("Invalid Interaction", () => {
    // TODO
  });

  describe("Ok", () => {
    it("Invoke `post_vaa`", async () => {
      const [signatureSets, args] = await defaultArgs(connection, payer);
    });
  });
});

type SignatureSets = {
  signatureSet: anchor.web3.Keypair;
  forkSignatureSet: anchor.web3.Keypair;
};

async function defaultArgs(
  connection: anchor.web3.Connection,
  payer: anchor.web3.Keypair
): Promise<[SignatureSets, coreBridge.LegacyPostVaaArgs]> {
  const signedVaa = defaultVaa();

  const [signatureSet, forkSignatureSet] = await parallelVerifySignatures(
    connection,
    payer,
    signedVaa
  );

  const parsed = parseVaa(signedVaa);

  return [
    {
      signatureSet,
      forkSignatureSet,
    },
    {
      version: parsed.version,
      guardianSetIndex: parsed.guardianSetIndex,
      timestamp: parsed.timestamp,
      nonce: parsed.nonce,
      emitterChain: parsed.emitterChain,
      emitterAddress: Array.from(parsed.emitterAddress),
      sequence: new anchor.BN(parsed.sequence.toString()),
      consistencyLevel: parsed.consistencyLevel,
      payload: parsed.payload,
    },
  ];
}

function defaultVaa(
  nonce?: number,
  payload?: Buffer,
  consistencyLevel?: number,
  timestamp?: number,
  guardianIndices?: number[]
) {
  if (nonce === undefined) {
    nonce = 420;
  }

  if (payload === undefined) {
    payload = Buffer.from("All your base are belong to us.");
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

async function parallelIxDetails(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: coreBridge.LegacyInitializeContext,
  args: coreBridge.LegacyInitializeArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  // const ix = coreBridge.legacyInitializeIx(program, accounts, args);

  // const forkedIx = coreBridge.legacyInitializeIx(forkedProgram, accounts, args);
  // return Promise.all([
  //   expectIxOkDetails(connection, [ix], [payer]),
  //   expectIxOkDetails(connection, [forkedIx], [payer]),
  // ]);
}
