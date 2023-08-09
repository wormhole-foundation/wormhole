import * as anchor from "@coral-xyz/anchor";
import { ethers } from "ethers";
import {
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  InvalidArgConfig,
  expectDeepEqual,
  expectIxErr,
  expectIxOkDetails,
  verifySignaturesAndPostVaa,
} from "../helpers";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import { transferMessageFeeIx } from "../helpers/coreBridge/utils";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";
import { createSetFeesInstruction } from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";

const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 2_003_000;

describe("Core Bridge -- Instruction: Set Message Fee", () => {
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
  const governance = new GovernanceEmitter(
    GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
    GOVERNANCE_SEQUENCE
  );
  const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

  describe("Invalid Interaction", () => {});

  describe("Ok", () => {
    it("Invoke `setMessageFee`", async () => {
      const amount = new anchor.BN(6969);

      const timestamp = 12345678;
      const chain = 1;
      const published = governance.publishWormholeSetMessageFee(
        timestamp,
        chain,
        BigInt(amount.toString())
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Verify and post
      await verifySignaturesAndPostVaa(program, payer, signedVaa);
      await verifySignaturesAndPostVaa(forkedProgram, payer, signedVaa);

      // Create the set message fee instruction.
      // const ix = coreBridge.legacySetMessageFeeIx(program);

      // Set message fee.
      // await expectIxOkDetails(
      //   connection,
      //   [createSetFeesInstruction(program.programId, payer, signedVaa)],
      //   [payer]
      // );
    });
  });
});

// function defaultArgs() {
//   return {
//     nonce: 420,
//     payload: Buffer.from("All your base are belong to us."),
//     finality: 1,
//   };
// }

// async function parallelTxDetails(
//   program: coreBridge.CoreBridgeProgram,
//   forkedProgram: coreBridge.CoreBridgeProgram,
//   accounts: coreBridge.LegacyPostMessageContext,
//   args: coreBridge.LegacyPostMessageArgs,
//   payer: anchor.web3.Keypair,
//   emitterSigner: anchor.web3.Keypair
// ) {
//   const connection = program.provider.connection;

//   // Create a message signer for both the main and forked program.
//   const messageSigner = anchor.web3.Keypair.generate();
//   const forkedMessageSigner = anchor.web3.Keypair.generate();

//   // Create the post message instruction.
//   accounts.message = messageSigner.publicKey;
//   const ix = coreBridge.legacyPostMessageIx(program, accounts, args);

//   // Create the post message instruction for the forked program.
//   accounts.message = forkedMessageSigner.publicKey;
//   const forkedIx = coreBridge.legacyPostMessageIx(forkedProgram, accounts, args);

//   // Pay the fee collector prior to publishing each message.
//   await Promise.all([
//     expectIxOkDetails(connection, [await transferMessageFeeIx(program, payer.publicKey)], [payer]),
//     expectIxOkDetails(
//       connection,
//       [await transferMessageFeeIx(forkedProgram, payer.publicKey)],
//       [payer]
//     ),
//   ]);

//   return Promise.all([
//     expectIxOkDetails(connection, [ix], [payer, emitterSigner, messageSigner]),
//     expectIxOkDetails(connection, [forkedIx], [payer, emitterSigner, forkedMessageSigner]),
//     messageSigner,
//     forkedMessageSigner,
//   ]);
// }
