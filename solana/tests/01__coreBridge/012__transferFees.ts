import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  ETHEREUM_DEADBEEF_TOKEN_ADDRESS,
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  createIfNeeded,
  createInvalidCoreGovernanceVaaFromEth,
  expectIxErr,
  expectIxOkDetails,
  invokeVerifySignaturesAndPostVaa,
  parallelPostVaa,
  expectIxOk,
  GOVERNANCE_EMITTER_ADDRESS,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 1_012_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// Test variables.
const localVariables = new Map<string, any>();

describe("Core Bridge -- Legacy Instruction: Transfer Fees", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  describe("Invalid Interaction", () => {
    const accountConfigs: InvalidAccountConfig[] = [
      {
        label: "config",
        contextName: "config",
        errorMsg: "ConstraintSeeds",
        dataLength: 24,
        owner: program.programId,
      },
      {
        label: "claim",
        contextName: "claim",
        errorMsg: "ConstraintSeeds",
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const accounts = await createIfNeeded(program.provider.connection, cfg, payer, {
          payer: payer.publicKey,
          recipient: payer.publicKey,
        } as coreBridge.LegacyTransferFeesContext);

        const signedVaa = defaultVaa(new anchor.BN(69), payer.publicKey);
        await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

        await expectIxErr(
          connection,
          [coreBridge.legacyTransferFeesIx(program, accounts, parseVaa(signedVaa))],
          [payer],
          cfg.errorMsg
        );
      });
    }
  });

  describe("Ok", () => {
    it("Invoke `transfer_fees`", async () => {
      // Amount of fees to be transferred and the recipient.
      const amount = 42069420;
      const recipient = anchor.web3.Keypair.generate().publicKey;

      // Check the balance of the recipient before the transfer.
      {
        const balance = await connection.getBalance(recipient);
        expect(balance).equals(0);
      }

      // Invoke the instruction.
      const signedVaa = await parallelTxDetails(
        program,
        forkedProgram,
        {
          payer: payer.publicKey,
          recipient,
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
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData).is.not.null;
      const forkFeeCollectorData = await connection.getAccountInfo(
        coreBridge.feeCollectorPda(program.programId)
      );
      expect(feeCollectorData!.lamports).to.equal(forkFeeCollectorData!.lamports);

      // Save the signed VAA for later.
      localVariables.set("amount", amount);
      localVariables.set("signedVaa", signedVaa);
      localVariables.set("recipient", recipient);
    });
  });

  describe("New implementation", () => {
    it("Cannot Invoke `transfer_fees` with Same VAA", async () => {
      const amount = localVariables.get("amount") as number;
      const signedVaa = localVariables.get("signedVaa") as Buffer;
      const recipient = localVariables.get("recipient") as anchor.web3.PublicKey;

      const transferIx = anchor.web3.SystemProgram.transfer({
        fromPubkey: payer.publicKey,
        toPubkey: coreBridge.feeCollectorPda(program.programId),
        lamports: amount,
      });
      //await expectIxOk(connection, [transferIx], [payer]);

      await expectIxErr(
        connection,
        [
          transferIx,
          coreBridge.legacyTransferFeesIx(
            program,
            { payer: payer.publicKey, recipient },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "already in use"
      );
    });

    it("Cannot Invoke `transfer_fees` with Invalid Governance Emitter", async () => {
      // Create a bad governance emitter.
      const governance = new GovernanceEmitter(
        Buffer.from(ETHEREUM_DEADBEEF_TOKEN_ADDRESS).toString("hex"),
        GOVERNANCE_SEQUENCE - 1
      );
      const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

      // Vaa info.
      const timestamp = 12345678;
      const chain = 1;
      const published = governance.publishWormholeTransferFees(
        timestamp,
        chain,
        BigInt(69),
        payer.publicKey.toBuffer()
      );
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and transfer fees.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyTransferFeesIx(
        program,
        { payer: payer.publicKey, recipient: payer.publicKey },
        parsedVaa
      );

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceEmitter");
    });

    it("Cannot Invoke `transfer_fees` with Invalid Governance Action", async () => {
      // Vaa info.
      const timestamp = 12345678;
      const chain = 1;

      // Publish the wrong VAA type.
      const published = governance.publishWormholeSetMessageFee(timestamp, chain, BigInt(69));

      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and transfer fees.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyTransferFeesIx(
        program,
        { payer: payer.publicKey, recipient: payer.publicKey },
        parsedVaa
      );

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceAction");
    });

    it("Cannot Invoke `transfer_fees` with Invalid Governance Vaa", async () => {
      const signedVaa = createInvalidCoreGovernanceVaaFromEth(
        guardians,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
        GOVERNANCE_SEQUENCE + 200,
        {
          governanceModule: Buffer.from(
            "00000000000000000000000000000000000000000000000000000000deadbeef",
            "hex"
          ),
        }
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyTransferFeesIx(
        program,
        { payer: payer.publicKey, recipient: payer.publicKey },
        parsedVaa
      );

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceVaa");
    });

    it("Cannot Invoke `transfer_fees` with Invalid Target Chain", async () => {
      // Fetch the default VAA.
      const invalidTargetChain = 69;
      const signedVaa = defaultVaa(new anchor.BN(69), payer.publicKey, invalidTargetChain);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and transfer fees.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyTransferFeesIx(
        program,
        { payer: payer.publicKey, recipient: payer.publicKey },
        parsedVaa
      );

      await expectIxErr(connection, [ix], [payer], "GovernanceForAnotherChain");
    });

    it("Cannot Invoke `transfer_fees` with Fee Larger than Max(u64)", async () => {
      // Fetch the default VAA.
      const signedVaa = defaultVaa(
        new anchor.BN(Buffer.from("010000000000000000", "hex")),
        payer.publicKey
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and transfer fees.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyTransferFeesIx(
        program,
        { payer: payer.publicKey, recipient: payer.publicKey },
        parsedVaa
      );

      await expectIxErr(connection, [ix], [payer], "U64Overflow");
    });

    it("Cannot Invoke `transfer_fees` with Fee Larger than Minimum Required Rent Balance", async () => {
      const lamportBalance = await connection
        .getAccountInfo(coreBridge.feeCollectorPda(program.programId))
        .then((info) => info!.lamports);

      // Fetch the default VAA.
      const signedVaa = defaultVaa(new anchor.BN(lamportBalance), payer.publicKey);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and transfer fees.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacyTransferFeesIx(
        program,
        { payer: payer.publicKey, recipient: payer.publicKey },
        parsedVaa
      );

      await expectIxErr(connection, [ix], [payer], "NotEnoughLamports");
    });

    it("Cannot Invoke `transfer_fees` With Invalid Recipient", async () => {
      // Invalid recipient.
      const recipient = anchor.web3.Keypair.generate();

      // Create a signed vaa with the payer as an invalid recipient.
      const signedVaa = defaultVaa(new anchor.BN(69), payer.publicKey);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Invoke the instruction.
      await expectIxErr(
        connection,
        [
          coreBridge.legacyTransferFeesIx(
            program,
            { payer: payer.publicKey, recipient: recipient.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "InvalidFeeRecipient"
      );
    });
  });
});

function defaultVaa(
  amount: anchor.BN,
  recipient: anchor.web3.PublicKey,
  targetChain?: number
): Buffer {
  const timestamp = 12345678;
  const chain = targetChain === undefined ? 1 : targetChain;
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
      toPubkey: coreBridge.feeCollectorPda(_program.programId),
      lamports: amount.toNumber(),
    });
    await expectIxOkDetails(connection, [transferIx], [payer]);
  }

  // Create the signed VAA.
  const signedVaa = defaultVaa(amount, accounts.recipient);
  const parsedVaa = parseVaa(signedVaa);

  // Post the VAAs.
  await parallelPostVaa(connection, payer, signedVaa);

  // Create the transferFees instruction.
  const ix = coreBridge.legacyTransferFeesIx(program, accounts, parsedVaa);
  const forkedIx = coreBridge.legacyTransferFeesIx(forkedProgram, accounts, parsedVaa);

  // Invoke the instruction.
  await expectIxOk(connection, [ix, forkedIx], [payer]);
  return signedVaa;
}
