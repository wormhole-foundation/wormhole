import { parseVaa } from "@certusone/wormhole-sdk";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import * as anchor from "@coral-xyz/anchor";
import { expect } from "chai";
import {
  ETHEREUM_TOKEN_ADDRESS_MAX_TWO,
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  createIfNeeded,
  expectIxErr,
  expectIxOk,
  invokeVerifySignaturesAndPostVaa,
  parallelPostVaa,
  createInvalidCoreGovernanceVaaFromEth,
  GOVERNANCE_EMITTER_ADDRESS,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 1_010_000;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Core Bridge -- Legacy Instruction: Set Message Fee", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  // Test variables.
  const localVariables = new Map<string, any>();

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
        } as coreBridge.LegacySetMessageFeeContext);

        const signedVaa = defaultVaa(new anchor.BN(69));
        await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

        await expectIxErr(
          connection,
          [coreBridge.legacySetMessageFeeIx(program, accounts, parseVaa(signedVaa))],
          [payer],
          cfg.errorMsg
        );
      });
    }
  });

  describe("Ok", () => {
    it("Invoke `set_message_fee`", async () => {
      // New fee amount.
      const amount = new anchor.BN(6969);

      // Fetch the bridge data before executing the instruction to verify that the
      // new fee amount is different than the current fee amount.
      const bridgeDataBefore = await coreBridge.Config.fromPda(connection, program.programId);
      expect(bridgeDataBefore.feeLamports.toString()).to.not.equal(amount.toString());

      // Fetch default VAA.
      const signedVaa = defaultVaa(amount);

      // Set the message fee for both programs.
      await parallelTxOk(program, forkedProgram, { payer: payer.publicKey }, signedVaa, payer);

      // Make sure the bridge accounts are the same.
      await coreBridge.expectEqualBridgeAccounts(program, forkedProgram);

      // Verify that the message fee was set correctly. We only need to check one program
      // since we already verified that the bridge accounts are the same.
      const bridgeDataAfter = await coreBridge.Config.fromPda(connection, program.programId);
      expect(bridgeDataAfter.feeLamports.toString()).to.equal(amount.toString());

      // Save the VAA.
      localVariables.set("signedVaa", signedVaa);
    });
  });

  describe("New implementation", () => {
    it("Cannot Invoke `set_message_fee` with Same VAA", async () => {
      const signedVaa = localVariables.get("signedVaa") as Buffer;

      await expectIxErr(
        connection,
        [
          coreBridge.legacySetMessageFeeIx(
            program,
            { payer: payer.publicKey },
            parseVaa(signedVaa)
          ),
        ],
        [payer],
        "already in use"
      );
    });

    it("Invoke `set_message_fee` with Claim Account Having Lamports", async () => {
      // New fee amount.
      const amount = new anchor.BN(6969);

      // Fetch default VAA.
      const signedVaa = defaultVaa(amount);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and update the message fee.
      const parsedVaa = parseVaa(signedVaa);

      // Find the claim account and send some lamports to it.
      const claim = coreBridge.Claim.address(
        program.programId,
        Array.from(parsedVaa.emitterAddress),
        parsedVaa.emitterChain,
        new anchor.BN(parsedVaa.sequence.toString())
      );

      const transferIx = anchor.web3.SystemProgram.transfer({
        fromPubkey: payer.publicKey,
        toPubkey: claim,
        lamports: 69,
      });

      // Create the instruction.
      const ix = coreBridge.legacySetMessageFeeIx(
        program,
        { payer: payer.publicKey, claim },
        parsedVaa
      );

      await expectIxOk(connection, [transferIx, ix], [payer]);
    });

    it("Cannot Invoke `set_message_fee` with Invalid Governance Emitter", async () => {
      // Create a bad governance emitter.
      const governance = new GovernanceEmitter(
        Buffer.from(ETHEREUM_TOKEN_ADDRESS_MAX_TWO).toString("hex"),
        GOVERNANCE_SEQUENCE - 1
      );
      const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

      // Vaa info.
      const timestamp = 12345678;
      const chain = 1;

      const published = governance.publishWormholeSetMessageFee(timestamp, chain, BigInt(69));
      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and update the message fee.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacySetMessageFeeIx(program, { payer: payer.publicKey }, parsedVaa);

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceEmitter");
    });

    it("Cannot Invoke `set_message_fee` with Invalid Governance Action", async () => {
      // Vaa info.
      const timestamp = 12345678;
      const chain = 1;

      // Publish the wrong VAA type.
      const published = governance.publishWormholeTransferFees(
        timestamp,
        chain,
        BigInt(69),
        Buffer.from(GOVERNANCE_EMITTER_ADDRESS.toString())
      );

      const signedVaa = guardians.addSignatures(
        published,
        [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]
      );

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and update the message fee.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacySetMessageFeeIx(program, { payer: payer.publicKey }, parsedVaa);

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceAction");
    });

    it("Cannot Invoke `set_message_fee` with Invalid Target Chain", async () => {
      // Fetch the default VAA.
      const invalidTargetChain = 69;
      const signedVaa = defaultVaa(new anchor.BN(69), invalidTargetChain);

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and update the message fee.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacySetMessageFeeIx(program, { payer: payer.publicKey }, parsedVaa);

      await expectIxErr(connection, [ix], [payer], "GovernanceForAnotherChain");
    });

    it("Cannot Invoke `set_message_fee` with Invalid Governance Vaa", async () => {
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

      // Parse the vaa and update the message fee.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacySetMessageFeeIx(program, { payer: payer.publicKey }, parsedVaa);

      await expectIxErr(connection, [ix], [payer], "InvalidGovernanceVaa");
    });

    it("Cannot Invoke `set_message_fee` with Fee Larger than Max(u64)", async () => {
      // Fetch the default VAA.
      const signedVaa = defaultVaa(new anchor.BN(Buffer.from("010000000000000000", "hex")));

      // Post the VAA.
      await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

      // Parse the vaa and update the message fee.
      const parsedVaa = parseVaa(signedVaa);

      // Create the instruction.
      const ix = coreBridge.legacySetMessageFeeIx(program, { payer: payer.publicKey }, parsedVaa);

      await expectIxErr(connection, [ix], [payer], "U64Overflow");
    });

    it("Invoke `post_message` with a Null Fee Collector (fee == 0)", async () => {
      // Fetch the initial fee state.
      const startingFeeLamports = await coreBridge.Config.fromPda(
        connection,
        program.programId
      ).then((data) => data.feeLamports);

      // Set the message fee to zero.
      await updateMessageFee(new anchor.BN(0), payer, program);

      // Post the message with a null fee collector.
      const message = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: message.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
        feeCollector: null,
      };

      // Post a message with a null fee collector.
      const ix = coreBridge.legacyPostMessageIx(program, accounts, {
        nonce: 420,
        payload: Buffer.from("All your base are belong to us."),
        commitment: "finalized",
      });
      await expectIxOk(connection, [ix], [payer, emitter, message]);

      // Set the message fee back to the original amount.
      await updateMessageFee(startingFeeLamports, payer, program);
    });

    it("Invoke `post_message_unreliable` with a Null Fee Collector (fee == 0)", async () => {
      // Fetch the initial fee state.
      const startingFeeLamports = await coreBridge.Config.fromPda(
        connection,
        program.programId
      ).then((data) => data.feeLamports);

      // Set the message fee to zero.
      await updateMessageFee(new anchor.BN(0), payer, program);

      // Post the message with a null fee collector.
      const message = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: message.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
        feeCollector: null,
      };

      // Post a message with a null fee collector.
      const ix = coreBridge.legacyPostMessageUnreliableIx(program, accounts, {
        nonce: 420,
        payload: Buffer.from("All your base are belong to us."),
        commitment: "finalized",
      });
      await expectIxOk(connection, [ix], [payer, emitter, message]);

      // Set the message fee back to the original amount.
      await updateMessageFee(startingFeeLamports, payer, program);
    });

    it("Cannot Invoke `post_message` with a Null Fee Collector (Fee > 0)", async () => {
      // Fetch the initial fee state.
      const startingFeeLamports = await coreBridge.Config.fromPda(
        connection,
        program.programId
      ).then((data) => data.feeLamports);
      expect(startingFeeLamports.gte(new anchor.BN(0))).to.be.true;

      // Post the message with a null fee collector.
      const message = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: message.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
        feeCollector: null,
      };

      // Post a message with a null fee collector.
      const ix = coreBridge.legacyPostMessageUnreliableIx(program, accounts, {
        nonce: 420,
        payload: Buffer.from("All your base are belong to us."),
        commitment: "finalized",
      });
      await expectIxErr(connection, [ix], [payer, emitter, message], "AccountNotEnoughKeys");
    });

    it("Cannot Invoke `post_message_unreliable` with a Null Fee Collector (Fee > 0)", async () => {
      // Fetch the initial fee state.
      const startingFeeLamports = await coreBridge.Config.fromPda(
        connection,
        program.programId
      ).then((data) => data.feeLamports);
      expect(startingFeeLamports.gte(new anchor.BN(0))).to.be.true;

      // Post the message with a null fee collector.
      const message = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: message.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
        feeCollector: null,
      };

      // Post a message with a null fee collector.
      const ix = coreBridge.legacyPostMessageIx(program, accounts, {
        nonce: 420,
        payload: Buffer.from("All your base are belong to us."),
        commitment: "finalized",
      });
      await expectIxErr(connection, [ix], [payer, emitter, message], "AccountNotEnoughKeys");
    });
  });
});

function defaultVaa(amount: anchor.BN, emitter?: number): Buffer {
  // Vaa info.
  const timestamp = 12345678;
  const chain = emitter === undefined ? 1 : emitter;

  const published = governance.publishWormholeSetMessageFee(
    timestamp,
    chain,
    BigInt(amount.toString())
  );
  return guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function updateMessageFee(
  amount: anchor.BN,
  payer: anchor.web3.Keypair,
  program: coreBridge.CoreBridgeProgram
) {
  // Set the message fee to zero.
  const timestamp = 12345678;
  const chain = 1;
  const published = governance.publishWormholeSetMessageFee(
    timestamp,
    chain,
    BigInt(amount.toString())
  );
  const signedVaa = guardians.addSignatures(published, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);

  // Post the VAA.
  await invokeVerifySignaturesAndPostVaa(program, payer, signedVaa);

  // Parse the vaa and update the message fee.
  const parsedVaa = parseVaa(signedVaa);

  // Create the instruction.
  const ix = coreBridge.legacySetMessageFeeIx(program, { payer: payer.publicKey }, parsedVaa);

  await expectIxOk(program.provider.connection, [ix], [payer]);
}

async function parallelTxOk(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  accounts: coreBridge.LegacySetMessageFeeContext,
  signedVaa: Buffer,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;

  // Post the VAAs.
  await parallelPostVaa(connection, payer, signedVaa);

  // Parse the VAA.
  const parsedVaa = parseVaa(signedVaa);

  // Create the set fee instructions.
  const ix = coreBridge.legacySetMessageFeeIx(program, accounts, parsedVaa);
  const forkedIx = coreBridge.legacySetMessageFeeIx(forkedProgram, accounts, parsedVaa);
  return expectIxOk(connection, [ix, forkedIx], [payer]);
}
