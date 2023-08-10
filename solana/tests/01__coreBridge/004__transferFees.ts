import * as anchor from "@coral-xyz/anchor";
import {
  GUARDIAN_KEYS,
  expectIxErr,
  expectIxOkDetails,
  InvalidAccountConfig,
  verifySignaturesAndPostVaa,
} from "../helpers";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";
import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 2_004_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Instruction: Transfer Fees", () => {
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

  describe("Invalid Interaction", () => {});

  describe("Ok", () => {
    it("Invoke `transferFees`", async () => {
      // Amount of fees to be transferred and the recipient.
      const amount = 42069420;
      const recipient = anchor.web3.Keypair.generate().publicKey;

      // Check the balance of the recipient before the transfer.
      {
        const balance = await connection.getBalance(recipient);
        expect(balance).equals(0);
      }

      // Invoke the instruction.
      const [txDetails, txForkDetails, signedVaa] = await parallelTxDetails(
        program,
        forkedProgram,
        {
          payer: payer.publicKey,
          recipient: recipient,
        },
        new anchor.BN(amount),
        payer
      );

      // Check the balance of the recipient after the transfer. The balance
      // should be two times the amount, since both programs should have
      // transferred the fees to the same recipient.
      {
        const balance = await connection.getBalance(recipient);
        expect(balance).equals(amount * 2);
      }

      // Compare the bridge data.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Validate fee collector.
      const feeCollectorData = await connection.getAccountInfo(
        coreBridge.FeeCollector.address(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.FeeCollector.address(program.programId)
      );
      expect(feeCollectorData!.lamports).to.equal(forkFeeCollectorData!.lamports);

      // Save the signed VAA for later.
      localVariables.set("signedVaa", signedVaa);
    });

    it("Cannot Invoke `transferFees` with Same VAA", async () => {
      const signedVaa: Buffer = localVariables.get("signedVaa");

      await expectIxErr(
        connection,
        [
          coreBridge.legacyTransferFeesIx(
            program,
            { payer: payer.publicKey, recipient: anchor.web3.Keypair.generate().publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "already in use"
      );
    });
  });
});

function defaultVaa(amount: anchor.BN, recipient: anchor.web3.PublicKey): Buffer {
  const timestamp = 12345678;
  const chain = 1;
  const published = governance.publishWormholeTransferFees(
    timestamp,
    chain,
    BigInt(amount.toString()),
    recipient.toBuffer()
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: coreBridge.LegacyTransferFeesContext,
  amount: anchor.BN,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // First send lamports over to the fee collectors.
  for (const _program of [program, forkedProgram]) {
    const transferIx = anchor.web3.SystemProgram.transfer({
      fromPubkey: payer.publicKey,
      toPubkey: coreBridge.FeeCollector.address(_program.programId),
      lamports: amount.toNumber(),
    });
    await expectIxOkDetails(connection, [transferIx], [payer]);
  }

  // Create the signed VAA.
  const signedVaa = defaultVaa(amount, accounts.recipient);
  const parsedVaa = parseVaa(signedVaa);

  // Verify and Post
  await Promise.all([
    verifySignaturesAndPostVaa(program, payer, signedVaa),
    verifySignaturesAndPostVaa(forkedProgram, payer, signedVaa),
  ]);

  // Create the transferFees instruction.
  const ix = await coreBridge.legacyTransferFeesIx(program, accounts, parsedVaa);
  const forkedIx = await coreBridge.legacyTransferFeesIx(forkedProgram, accounts, parsedVaa);

  // Invoke the instruction.
  return Promise.all([
    expectIxOkDetails(connection, [ix], [payer]),
    expectIxOkDetails(connection, [forkedIx], [payer]),
    signedVaa,
  ]);
}
