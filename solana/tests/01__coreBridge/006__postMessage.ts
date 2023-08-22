import * as anchor from "@coral-xyz/anchor";
import { ethers } from "ethers";
import {
  InvalidAccountConfig,
  InvalidArgConfig,
  NullableAccountConfig,
  createAccountIx,
  createIfNeeded,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
  expectIxOkDetails,
  expectedPayerSequence,
  invokeVerifySignaturesAndPostVaa,
  GUARDIAN_KEYS,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import { expect } from "chai";
import { GovernanceEmitter, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import { GOVERNANCE_EMITTER_ADDRESS } from "../helpers/coreBridge";
import { parseVaa } from "@certusone/wormhole-sdk";

// Mock governance emitter and guardian.
const GUARDIAN_SET_INDEX = 0;
const GOVERNANCE_SEQUENCE = 999_999;
const governance = new GovernanceEmitter(
  GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex"),
  GOVERNANCE_SEQUENCE - 1
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);
describe("Core Bridge -- Legacy Instruction: Post Message", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;
  const forkedProgram = coreBridge.getAnchorProgram(connection, coreBridge.mainnet());

  const commonEmitterSequence = new anchor.BN(0);

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
        label: "fee_collector",
        contextName: "feeCollector",
        errorMsg: "ConstraintSeeds",
        dataLength: 0,
        owner: anchor.web3.PublicKey.default,
      },
      {
        label: "emitter_sequence",
        contextName: "emitterSequence",
        errorMsg: "ConstraintSeeds",
        dataLength: 8,
        owner: program.programId,
      },
    ];

    for (const cfg of accountConfigs) {
      it(`Account: ${cfg.label} (${cfg.errorMsg})`, async () => {
        const message = anchor.web3.Keypair.generate();
        const emitter = anchor.web3.Keypair.generate();
        const accounts = await createIfNeeded(program.provider.connection, cfg, payer, {
          message: message.publicKey,
          emitter: emitter.publicKey,
          payer: payer.publicKey,
        } as coreBridge.LegacyPostMessageContext);

        // Create the post message instruction.
        const ix = coreBridge.legacyPostMessageIx(program, accounts, defaultArgs());
        await expectIxErr(connection, [ix], [payer, emitter, message], cfg.errorMsg);
      });
    }
  });

  describe("Ok", () => {
    it("Invoke `post_message`", async () => {
      // Fetch default args.
      await parallelEverythingOk(
        program,
        forkedProgram,
        { payer, emitter: payer },
        defaultArgs(),
        commonEmitterSequence
      );
    });

    it("Invoke `post_message` Again With Same Emitter", async () => {
      // Fetch default args.
      const { nonce, finality } = defaultArgs();

      // Change the payload.
      const payload = Buffer.from("Somebody set up us the bomb.");

      await parallelEverythingOk(
        program,
        forkedProgram,
        { payer, emitter: payer },
        { nonce, finality, payload },
        commonEmitterSequence
      );
    });

    it("Invoke `post_message` (Emitter != Payer)", async () => {
      // Create new emitter.
      const emitter = anchor.web3.Keypair.generate();

      // Fetch default args.
      await parallelEverythingOk(
        program,
        forkedProgram,
        { payer, emitter },
        defaultArgs(),
        new anchor.BN(0)
      );
    });
  });

  describe("New implementation", () => {
    const nullableAccountConfigs: NullableAccountConfig[] = [
      {
        label: "rent",
        contextName: "rent",
      },
      {
        label: "clock",
        contextName: "clock",
      },
    ];

    for (const cfg of nullableAccountConfigs) {
      it(`Invoke \`post_message\` without Account: ${cfg.label}`, async () => {
        const emitter = anchor.web3.Keypair.generate();

        const nullAccounts = {
          feeCollector: false,
          clock: false,
          rent: false,
        };
        nullAccounts[cfg.contextName] = true;

        // Fetch default args.
        await parallelEverythingOk(
          program,
          forkedProgram,
          { payer, emitter },
          defaultArgs(),
          new anchor.BN(0),
          nullAccounts
        );
      });
    }

    it("Cannot Invoke `post_message` Without Paying Fee", async () => {
      // Create the post message instruction.
      const message = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: message.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
      };
      const ix = coreBridge.legacyPostMessageIx(program, accounts, defaultArgs());
      await expectIxErr(connection, [ix], [payer, emitter, message], "InsufficientFees");
    });

    it("Cannot Invoke `post_message` With Invalid Payload", async () => {
      // Create the post message instruction.
      const message = anchor.web3.Keypair.generate();
      const emitter = anchor.web3.Keypair.generate();
      const accounts = {
        message: message.publicKey,
        emitter: emitter.publicKey,
        payer: payer.publicKey,
      };
      let { nonce, payload, finality } = defaultArgs();
      payload = Buffer.alloc(0);

      const ix = coreBridge.legacyPostMessageIx(program, accounts, {
        nonce,
        payload,
        finality,
      });
      await expectIxErr(connection, [ix], [payer, emitter, message], "InvalidInstructionArgument");
    });
  });
});

function defaultArgs() {
  return {
    nonce: 420,
    payload: Buffer.from("All your base are belong to us."),
    finality: 1,
  };
}

async function everythingOk(
  program: coreBridge.CoreBridgeProgram,
  signers: {
    payer: anchor.web3.Keypair;
    emitter: anchor.web3.Keypair;
  },
  args: coreBridge.LegacyPostMessageArgs,
  sequence: anchor.BN,
  nullAccounts?: { feeCollector: boolean; clock: boolean; rent: boolean }
) {
  const { payer, emitter } = signers;
  const message = anchor.web3.Keypair.generate();

  const out = await coreBridge.expectOkPostMessage(
    program,
    { payer, emitter, message },
    args,
    sequence,
    args.payload,
    nullAccounts
  );

  sequence.iaddn(1);

  return out;
}

async function parallelEverythingOk(
  program: coreBridge.CoreBridgeProgram,
  forkedProgram: coreBridge.CoreBridgeProgram,
  signers: {
    payer: anchor.web3.Keypair;
    emitter: anchor.web3.Keypair;
  },
  args: coreBridge.LegacyPostMessageArgs,
  sequence: anchor.BN,
  nullAccounts?: { feeCollector: boolean; clock: boolean; rent: boolean }
) {
  const { payer, emitter } = signers;
  const message = anchor.web3.Keypair.generate();
  const forkMessage = anchor.web3.Keypair.generate();

  const [out, forkOut] = await Promise.all([
    coreBridge.expectOkPostMessage(
      program,
      { payer, emitter, message },
      args,
      sequence,
      args.payload,
      nullAccounts
    ),
    coreBridge.expectOkPostMessage(
      forkedProgram,
      { payer, emitter, message: forkMessage },
      args,
      sequence,
      args.payload
    ),
  ]);

  for (const key of ["postedMessageData", "emitterSequence", "config"]) {
    expectDeepEqual(out[key], forkOut[key]);
  }

  sequence.iaddn(1);
}
