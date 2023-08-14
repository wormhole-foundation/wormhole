import { MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { ethers } from "ethers";
import { GUARDIAN_KEYS, createSecp256k1Instruction, expectDeepEqual, expectIxOk } from "../helpers";
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
    // TODO

    it.skip("Cannot Invoke `verify_signatures` with Different Message on Existing Signature Set", async () => {
      // TODO
    });

    it.skip("Cannot Invoke `verify_signatures` with Signer Indices Mismatch", async () => {
      // tODO
    });
  });

  describe("Ok", () => {
    const message = Buffer.from("Not a Wormhole message.");
    const messageHash = Buffer.from(ethers.utils.arrayify(ethers.utils.keccak256(message)));

    // This signature set will be written multiple times.
    const signatureSet = anchor.web3.Keypair.generate();

    for (let i = 0; i < 17; i += 2) {
      it(`Invoke \`verify_signatures\` for Guardians [${i}], [${i + 1}], [${i + 2}]`, async () => {
        const guardianSet = coreBridge.GuardianSet.address(program.programId, GUARDIAN_SET_INDEX);
        const sigVerifyIx = await createSigVerifyIx(program, GUARDIAN_SET_INDEX, message, [
          i,
          i + 1,
          i + 2,
        ]);
        const signerIndices = new Array(19).fill(-1);
        signerIndices[i] = 0;
        signerIndices[i + 1] = 1;
        signerIndices[i + 2] = 2;

        const verifyIx = coreBridge.legacyVerifySignaturesIx(
          program,
          { payer: payer.publicKey, guardianSet, signatureSet: signatureSet.publicKey },
          {
            signerIndices,
          }
        );
        await expectIxOk(connection, [sigVerifyIx, verifyIx], [payer, signatureSet]);

        const signatureSetData = await coreBridge.SignatureSet.fromAccountAddress(
          connection,
          signatureSet.publicKey
        );
        const sigVerifySuccesses: boolean[] = new Array(19).fill(false);
        for (let j = 0; j <= i + 2; ++j) {
          sigVerifySuccesses[j] = true;
        }
        expectDeepEqual(signatureSetData, {
          sigVerifySuccesses,
          messageHash: Array.from(messageHash),
          guardianSetIndex: GUARDIAN_SET_INDEX,
        });
      });
    }

    it.skip("Invoke `verify_signatures` with Same Guardian on Existing Signature Set", async () => {
      // TODO
    });
  });
});

function generateSignature(message: Buffer, guardianIndex: number) {
  return guardians.addSignatures(message, [guardianIndex]).subarray(7, 7 + 65);
}

async function createSigVerifyIx(
  program: coreBridge.CoreBridgeProgram,
  guardianSetIndex: number,
  message: Buffer,
  guardianIndices: number[]
) {
  const guardianSet = coreBridge.GuardianSet.address(program.programId, guardianSetIndex);
  const ethAddresses = await coreBridge.GuardianSet.fromAccountAddress(
    program.provider.connection,
    guardianSet
  ).then((acct) => guardianIndices.map((i) => Buffer.from(acct.keys[i])));
  const signatures = guardianIndices.map((i) => generateSignature(message, i));

  return createSecp256k1Instruction(
    signatures,
    ethAddresses,
    Buffer.from(ethers.utils.arrayify(ethers.utils.keccak256(message)))
  );
}

function defaultArgs() {
  return {};
}

async function parallelIxOk(
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
