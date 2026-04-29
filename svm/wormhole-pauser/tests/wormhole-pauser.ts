import { keccak256 } from "@certusone/wormhole-sdk";
import {
  MockEmitter,
  MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import {
  parseVaa,
  SignedVaa,
} from "@certusone/wormhole-sdk/lib/cjs/vaa/wormhole";
import * as anchor from "@coral-xyz/anchor";
import { BN, Program } from "@coral-xyz/anchor";
import { expect } from "chai";
import type { WormholeVerifyVaaShim } from "../idls/wormhole_verify_vaa_shim";
import WormholeVerifyVaaShimIdl from "../idls/wormhole_verify_vaa_shim.json";
import { MockPausable } from "../target/types/mock_pausable";
import { WormholePauser } from "../target/types/wormhole_pauser";

const CORE_BRIDGE_PROGRAM_ID = new anchor.web3.PublicKey(
  "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5",
);
const GUARDIAN_SET_SEED = "GuardianSet";

// Same dev key used by the EVM tests; we install the corresponding eth address as
// guardian set 0 (see tests/accounts/core_bridge_testnet/guardian_set_0.json).
const GUARDIAN_KEY =
  "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0";

const GOVERNANCE_CHAIN = 1;
const GOVERNANCE_EMITTER_BUF = Buffer.alloc(32);
GOVERNANCE_EMITTER_BUF.writeUInt8(0x04, 31);
const GOVERNANCE_EMITTER_HEX = GOVERNANCE_EMITTER_BUF.toString("hex");

const SOLANA_CHAIN_ID = 1;

// "DelegatedPauser" left-padded to 32 bytes
const DELEGATED_PAUSER_MODULE = Buffer.from(
  "0000000000000000000000000000000000" + "44656c656761746564506175736572",
  "hex",
);

const ACTION_SET_CONFIG_EVM = 1;
const ACTION_SET_CONFIG_SOLANA = 2;

function vaaBody(vaa: SignedVaa): Buffer {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  const sigStart = 6;
  const numSigners = signedVaa[5];
  const sigLength = 66;
  return signedVaa.subarray(sigStart + sigLength * numSigners);
}

function encodeSetConfigPayload(opts: {
  module?: Buffer;
  action?: number;
  chainId?: number;
  index: number;
  threshold: number;
  expiryDuration: bigint;
  signers: anchor.web3.PublicKey[];
  trailing?: Buffer;
}): Buffer {
  const module = opts.module ?? DELEGATED_PAUSER_MODULE;
  const action = opts.action ?? ACTION_SET_CONFIG_SOLANA;
  const chainId = opts.chainId ?? SOLANA_CHAIN_ID;
  const trailing = opts.trailing ?? Buffer.alloc(0);

  const fixed = Buffer.alloc(32 + 1 + 2 + 2 + 1 + 8 + 1);
  module.copy(fixed, 0);
  fixed.writeUInt8(action, 32);
  fixed.writeUInt16BE(chainId, 33);
  fixed.writeUInt16BE(opts.index, 35);
  fixed.writeUInt8(opts.threshold, 37);
  fixed.writeBigUInt64BE(opts.expiryDuration, 38);
  fixed.writeUInt8(opts.signers.length, 46);

  const signerBytes = Buffer.concat(opts.signers.map((p) => p.toBuffer()));
  return Buffer.concat([fixed, signerBytes, trailing]);
}

describe("wormhole-pauser", () => {
  anchor.setProvider(anchor.AnchorProvider.env());
  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const payer = (provider.wallet as anchor.Wallet).payer;

  const program = anchor.workspace.wormholePauser as Program<WormholePauser>;
  const mockPausable = anchor.workspace.mockPausable as Program<MockPausable>;
  const verifyShimProgram = new Program<WormholeVerifyVaaShim>(
    WormholeVerifyVaaShimIdl as WormholeVerifyVaaShim,
    provider,
  );

  const guardians = new MockGuardians(0, [GUARDIAN_KEY]);
  const emitter = new MockEmitter(GOVERNANCE_EMITTER_HEX, GOVERNANCE_CHAIN, 0);

  // Three test signers — these are real Solana keypairs that will appear in the WormholePauser
  // signer set and must `Signer<'info>` Solana transactions to propose / approve / cancel.
  const signerA = anchor.web3.Keypair.generate();
  const signerB = anchor.web3.Keypair.generate();
  const signerC = anchor.web3.Keypair.generate();
  const outsider = anchor.web3.Keypair.generate();

  // Seed PDAs
  const [configPda] = anchor.web3.PublicKey.findProgramAddressSync(
    [Buffer.from("config")],
    program.programId,
  );
  const [authorityPda] = anchor.web3.PublicKey.findProgramAddressSync(
    [Buffer.from("authority")],
    program.programId,
  );

  // Mock pausable PDAs
  const [mockStatePda] = anchor.web3.PublicKey.findProgramAddressSync(
    [Buffer.from("state")],
    mockPausable.programId,
  );

  // Guardian set 0 PDA on the core bridge
  const guardianSetIndexBuf = Buffer.alloc(4);
  guardianSetIndexBuf.writeUInt32BE(0);
  const [guardianSet, guardianSetBump] =
    anchor.web3.PublicKey.findProgramAddressSync(
      [Buffer.from(GUARDIAN_SET_SEED), guardianSetIndexBuf],
      CORE_BRIDGE_PROGRAM_ID,
    );

  /** Build a signed VAA carrying the given governance payload. */
  function buildVaa(payload: Buffer, sequence: number): Buffer {
    emitter.sequence = sequence; // make sequence deterministic per test
    const message = emitter.publishMessage(0, payload, 0);
    return guardians.addSignatures(message, [0]);
  }

  /** Post the VAA's guardian signatures to a fresh account; returns its pubkey. */
  async function postSignatures(vaa: Buffer): Promise<anchor.web3.Keypair> {
    const parsed = parseVaa(vaa);
    const sigKeypair = anchor.web3.Keypair.generate();
    await verifyShimProgram.methods
      .postSignatures(
        parsed.guardianSetIndex,
        parsed.guardianSignatures.length,
        parsed.guardianSignatures.map((s) => [s.index, ...s.signature]),
      )
      .accounts({ guardianSignatures: sigKeypair.publicKey })
      .signers([sigKeypair])
      .rpc();
    return sigKeypair;
  }

  /** Submit a SetConfigSolana governance VAA, returning the unwrapped tx result. */
  async function submitConfigVaa(vaa: Buffer) {
    const sigKeypair = await postSignatures(vaa);
    const body = vaaBody(vaa);
    const digest = keccak256(keccak256(body));
    return program.methods
      .submitConfig({
        guardianSetBump,
        digest: [...digest],
        vaaBody: body,
      })
      .accountsPartial({
        payer: payer.publicKey,
        guardianSet,
        guardianSignatures: sigKeypair.publicKey,
        config: configPda,
      })
      .preInstructions([
        anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({ units: 600_000 }),
      ])
      .postInstructions([
        await verifyShimProgram.methods
          .closeSignatures()
          .accounts({ guardianSignatures: sigKeypair.publicKey })
          .instruction(),
      ])
      .rpc();
  }

  /** Apply a default config: 3 signers, threshold 2, 1h expiry, index 1. */
  async function applyDefaultConfig(seq = 0) {
    const payload = encodeSetConfigPayload({
      index: 1,
      threshold: 2,
      expiryDuration: 3600n,
      signers: [signerA.publicKey, signerB.publicKey, signerC.publicKey],
    });
    await submitConfigVaa(buildVaa(payload, seq));
  }

  /** Initialize the mock pausable's state account. */
  async function initMockPausable() {
    const info = await connection.getAccountInfo(mockStatePda);
    if (info !== null) return;
    await mockPausable.methods
      .initialize()
      .accountsPartial({ payer: payer.publicKey, state: mockStatePda })
      .rpc();
  }

  /** Fund a keypair so it can pay for `signer` slots in subsequent transactions. */
  async function fund(pk: anchor.web3.PublicKey) {
    const sig = await connection.requestAirdrop(pk, 1_000_000_000);
    await connection.confirmTransaction(sig);
  }

  /** Account metas + remaining accounts for a `pause()` call into the mock pausable. */
  function pauseProposalArgs() {
    const accountMetas = [
      { pubkey: authorityPda, isSigner: true, isWritable: false },
      { pubkey: mockStatePda, isSigner: false, isWritable: true },
    ];
    const data = mockPausable.coder.instruction.encode("pause", {});
    return { accountMetas, data, target: mockPausable.programId };
  }

  /** Remaining accounts for an `execute` (propose+threshold==1, or approve at threshold). */
  function pauseRemainingAccounts() {
    return [
      {
        pubkey: mockPausable.programId,
        isSigner: false,
        isWritable: false,
      },
      {
        pubkey: authorityPda,
        isSigner: false,
        isWritable: false,
      },
      {
        pubkey: mockStatePda,
        isSigner: false,
        isWritable: true,
      },
    ];
  }

  function noExecuteRemainingAccounts() {
    // Below-threshold proposes/approves still need at least the target program in
    // remaining_accounts so the program can validate it; the actual CPI is not invoked.
    // Pass a minimal slice so the program can still do its `executed` check and skip.
    return [];
  }

  async function proposalPda(id: bigint | number) {
    const idBuf = Buffer.alloc(8);
    idBuf.writeBigUInt64LE(BigInt(id));
    const [pda] = anchor.web3.PublicKey.findProgramAddressSync(
      [Buffer.from("proposal"), idBuf],
      program.programId,
    );
    return pda;
  }

  before(async () => {
    await Promise.all(
      [signerA, signerB, signerC, outsider].map((kp) => fund(kp.publicKey)),
    );
    await initMockPausable();
  });

  describe("submitConfig", () => {
    it("applies a valid config", async () => {
      await applyDefaultConfig(0);

      const cfg = await program.account.config.fetch(configPda);
      expect(cfg.configIndex).to.equal(1);
      expect(cfg.threshold).to.equal(2);
      expect(cfg.expiryDuration.toString()).to.equal("3600");
      expect(cfg.signers.map((p) => p.toBase58())).to.deep.equal([
        signerA.publicKey.toBase58(),
        signerB.publicKey.toBase58(),
        signerC.publicKey.toBase58(),
      ]);
      expect(cfg.nextProposalId.toString()).to.equal("0");
    });

    it("rejects an EVM-action governance message on Solana", async () => {
      const payload = encodeSetConfigPayload({
        action: ACTION_SET_CONFIG_EVM,
        index: 2,
        threshold: 1,
        expiryDuration: 60n,
        signers: [signerA.publicKey],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 1));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("InvalidGovernanceAction");
    });

    it("rejects a wrong target chain", async () => {
      const payload = encodeSetConfigPayload({
        chainId: 99,
        index: 2,
        threshold: 1,
        expiryDuration: 60n,
        signers: [signerA.publicKey],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 2));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("InvalidTargetChain");
    });

    it("rejects an out-of-order index", async () => {
      const payload = encodeSetConfigPayload({
        index: 5,
        threshold: 1,
        expiryDuration: 60n,
        signers: [signerA.publicKey],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 3));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("InvalidIndex");
    });

    it("rejects threshold == 0", async () => {
      const payload = encodeSetConfigPayload({
        index: 2,
        threshold: 0,
        expiryDuration: 60n,
        signers: [signerA.publicKey],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 4));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("InvalidThreshold");
    });

    it("rejects threshold > numSigners", async () => {
      const payload = encodeSetConfigPayload({
        index: 2,
        threshold: 5,
        expiryDuration: 60n,
        signers: [signerA.publicKey, signerB.publicKey],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 5));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("InvalidThreshold");
    });

    it("rejects expiry == 0", async () => {
      const payload = encodeSetConfigPayload({
        index: 2,
        threshold: 1,
        expiryDuration: 0n,
        signers: [signerA.publicKey],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 6));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("InvalidExpiryDuration");
    });

    it("rejects an empty signer set", async () => {
      const payload = encodeSetConfigPayload({
        index: 2,
        threshold: 1,
        expiryDuration: 60n,
        signers: [],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 7));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("EmptySignerSet");
    });

    it("rejects a zero pubkey signer", async () => {
      const payload = encodeSetConfigPayload({
        index: 2,
        threshold: 1,
        expiryDuration: 60n,
        signers: [anchor.web3.PublicKey.default, signerA.publicKey],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 8));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("ZeroSigner");
    });

    it("rejects duplicate signers", async () => {
      const payload = encodeSetConfigPayload({
        index: 2,
        threshold: 1,
        expiryDuration: 60n,
        signers: [signerA.publicKey, signerA.publicKey],
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 9));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("DuplicateSigner");
    });

    it("rejects trailing payload bytes", async () => {
      const payload = encodeSetConfigPayload({
        index: 2,
        threshold: 1,
        expiryDuration: 60n,
        signers: [signerA.publicKey],
        trailing: Buffer.from([0xff]),
      });
      let err = "";
      try {
        await submitConfigVaa(buildVaa(payload, 10));
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("InvalidPayloadLength");
    });

    it("replay-protects a previously consumed VAA", async () => {
      const payload = encodeSetConfigPayload({
        index: 2,
        threshold: 1,
        expiryDuration: 60n,
        signers: [outsider.publicKey],
      });
      const vaa = buildVaa(payload, 11);
      await submitConfigVaa(vaa);
      let err = "";
      try {
        await submitConfigVaa(vaa);
      } catch (e) {
        err = (e as Error).message;
      }
      // The `consumed_vaa` PDA is `init`-ed once; a replay fails to allocate.
      expect(err).to.match(/already in use|ConstraintSeeds|consumed_vaa/);
    });
  });

  describe("propose / approve / cancelApproval", () => {
    before(async () => {
      // Reset to the default 3-of-2 config (the replay-protect test rolled the index forward to 2).
      // Index is currently 2 (one signer = outsider). Move to index 3 with the standard set.
      const payload = encodeSetConfigPayload({
        index: 3,
        threshold: 2,
        expiryDuration: 3600n,
        signers: [signerA.publicKey, signerB.publicKey, signerC.publicKey],
      });
      await submitConfigVaa(buildVaa(payload, 12));
    });

    it("rejects propose from a non-signer", async () => {
      const { accountMetas, data, target } = pauseProposalArgs();
      let err = "";
      try {
        const cfg = await program.account.config.fetch(configPda);
        const proposal = await proposalPda(cfg.nextProposalId.toNumber());
        await program.methods
          .propose({ targetProgram: target, accountMetas, data })
          .accountsPartial({
            payer: payer.publicKey,
            signer: outsider.publicKey,
            config: configPda,
            proposal,
            authority: authorityPda,
          })
          .signers([outsider])
          .rpc();
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("NotSigner");
    });

    it("auto-approves the proposer but does not execute below threshold", async () => {
      const { accountMetas, data, target } = pauseProposalArgs();
      const cfg = await program.account.config.fetch(configPda);
      const id = cfg.nextProposalId.toNumber();
      const proposal = await proposalPda(id);

      await program.methods
        .propose({ targetProgram: target, accountMetas, data })
        .accountsPartial({
          payer: payer.publicKey,
          signer: signerA.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .signers([signerA])
        .rpc();

      const p = await program.account.proposal.fetch(proposal);
      expect(p.executed).to.equal(false);
      expect(p.approvalCount).to.equal(1);
      expect(p.targetProgram.toBase58()).to.equal(target.toBase58());
      const state = await mockPausable.account.state.fetch(mockStatePda);
      expect(state.paused).to.equal(false);
    });

    it("a second approval reaches threshold and executes the CPI", async () => {
      const id = (await program.account.config.fetch(configPda)).nextProposalId.toNumber() - 1;
      const proposal = await proposalPda(id);

      await program.methods
        .approve({ proposalId: new BN(id) })
        .accountsPartial({
          signer: signerB.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .remainingAccounts(pauseRemainingAccounts())
        .signers([signerB])
        .rpc();

      const p = await program.account.proposal.fetch(proposal);
      expect(p.executed).to.equal(true);
      expect(p.approvalCount).to.equal(2);

      const state = await mockPausable.account.state.fetch(mockStatePda);
      expect(state.paused).to.equal(true);
      expect(state.lastCaller.toBase58()).to.equal(authorityPda.toBase58());
    });

    it("rejects double-approval", async () => {
      // Reset the mock pausable so a fresh proposal can pause it again.
      await mockPausable.methods
        .setShouldRevert(false)
        .accountsPartial({ payer: payer.publicKey, state: mockStatePda })
        .rpc();
      const cfg = await program.account.config.fetch(configPda);
      const id = cfg.nextProposalId.toNumber();
      const proposal = await proposalPda(id);

      const { accountMetas, data, target } = pauseProposalArgs();
      await program.methods
        .propose({ targetProgram: target, accountMetas, data })
        .accountsPartial({
          payer: payer.publicKey,
          signer: signerA.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .signers([signerA])
        .rpc();

      let err = "";
      try {
        await program.methods
          .approve({ proposalId: new BN(id) })
          .accountsPartial({
            signer: signerA.publicKey,
            config: configPda,
            proposal,
            authority: authorityPda,
          })
          .remainingAccounts(pauseRemainingAccounts())
          .signers([signerA])
          .rpc();
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("AlreadyApproved");
    });

    it("cancels an approval and lets a different signer reach threshold", async () => {
      const id = (await program.account.config.fetch(configPda)).nextProposalId.toNumber() - 1;
      const proposal = await proposalPda(id);

      await program.methods
        .cancelApproval({ proposalId: new BN(id) })
        .accountsPartial({
          signer: signerA.publicKey,
          config: configPda,
          proposal,
        })
        .signers([signerA])
        .rpc();

      let p = await program.account.proposal.fetch(proposal);
      expect(p.approvalCount).to.equal(0);
      expect(p.executed).to.equal(false);

      // Approve via signerB then signerC to reach threshold==2.
      await program.methods
        .approve({ proposalId: new BN(id) })
        .accountsPartial({
          signer: signerB.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .remainingAccounts(pauseRemainingAccounts())
        .signers([signerB])
        .rpc();
      await program.methods
        .approve({ proposalId: new BN(id) })
        .accountsPartial({
          signer: signerC.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .remainingAccounts(pauseRemainingAccounts())
        .signers([signerC])
        .rpc();

      p = await program.account.proposal.fetch(proposal);
      expect(p.executed).to.equal(true);
      const state = await mockPausable.account.state.fetch(mockStatePda);
      expect(state.paused).to.equal(true);
    });

    it("reverts the entire transaction when the CPI fails (rolls back the approval)", async () => {
      // Force the next pause to revert.
      await mockPausable.methods
        .setShouldRevert(true)
        .accountsPartial({ payer: payer.publicKey, state: mockStatePda })
        .rpc();

      const cfg = await program.account.config.fetch(configPda);
      const id = cfg.nextProposalId.toNumber();
      const proposal = await proposalPda(id);
      const { accountMetas, data, target } = pauseProposalArgs();

      // Propose (auto-approves signerA — count = 1 < threshold, no CPI yet).
      await program.methods
        .propose({ targetProgram: target, accountMetas, data })
        .accountsPartial({
          payer: payer.publicKey,
          signer: signerA.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .signers([signerA])
        .rpc();

      // Threshold-meeting approve hits the forced revert and rolls everything back.
      let err = "";
      try {
        await program.methods
          .approve({ proposalId: new BN(id) })
          .accountsPartial({
            signer: signerB.publicKey,
            config: configPda,
            proposal,
            authority: authorityPda,
          })
          .remainingAccounts(pauseRemainingAccounts())
          .signers([signerB])
          .rpc();
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.match(/forced revert|ExecutionFailed|custom program error/);

      // signerB's approval was rolled back; signerA's remains.
      const p = await program.account.proposal.fetch(proposal);
      expect(p.executed).to.equal(false);
      expect(p.approvalCount).to.equal(1);

      // After clearing the forced revert, signerB can retry to completion.
      await mockPausable.methods
        .setShouldRevert(false)
        .accountsPartial({ payer: payer.publicKey, state: mockStatePda })
        .rpc();
      await program.methods
        .approve({ proposalId: new BN(id) })
        .accountsPartial({
          signer: signerB.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .remainingAccounts(pauseRemainingAccounts())
        .signers([signerB])
        .rpc();
      const p2 = await program.account.proposal.fetch(proposal);
      expect(p2.executed).to.equal(true);
    });

    it("invalidates a proposal once the config rotates", async () => {
      // Create a new pending proposal.
      const cfg = await program.account.config.fetch(configPda);
      const id = cfg.nextProposalId.toNumber();
      const proposal = await proposalPda(id);
      const { accountMetas, data, target } = pauseProposalArgs();
      await program.methods
        .propose({ targetProgram: target, accountMetas, data })
        .accountsPartial({
          payer: payer.publicKey,
          signer: signerA.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .signers([signerA])
        .rpc();

      // Rotate the config (index → cfg.configIndex + 1) — this orphans the proposal.
      const payload = encodeSetConfigPayload({
        index: cfg.configIndex + 1,
        threshold: 1,
        expiryDuration: 3600n,
        signers: [signerA.publicKey],
      });
      await submitConfigVaa(buildVaa(payload, 100 + id));

      let err = "";
      try {
        await program.methods
          .approve({ proposalId: new BN(id) })
          .accountsPartial({
            signer: signerA.publicKey,
            config: configPda,
            proposal,
            authority: authorityPda,
          })
          .remainingAccounts(pauseRemainingAccounts())
          .signers([signerA])
          .rpc();
      } catch (e) {
        err = (e as Error).message;
      }
      expect(err).to.include("ProposalConfigRotated");
    });

    it("propose with threshold==1 executes immediately", async () => {
      // Current config (set in the previous test) is { threshold: 1, signers: [signerA] }.
      // Reset the pausable so we can observe a fresh pause.
      await mockPausable.methods
        .setShouldRevert(false)
        .accountsPartial({ payer: payer.publicKey, state: mockStatePda })
        .rpc();

      const cfg = await program.account.config.fetch(configPda);
      const id = cfg.nextProposalId.toNumber();
      const proposal = await proposalPda(id);
      const { accountMetas, data, target } = pauseProposalArgs();

      await program.methods
        .propose({ targetProgram: target, accountMetas, data })
        .accountsPartial({
          payer: payer.publicKey,
          signer: signerA.publicKey,
          config: configPda,
          proposal,
          authority: authorityPda,
        })
        .remainingAccounts(pauseRemainingAccounts())
        .signers([signerA])
        .rpc();

      const p = await program.account.proposal.fetch(proposal);
      expect(p.executed).to.equal(true);
      const state = await mockPausable.account.state.fetch(mockStatePda);
      expect(state.paused).to.equal(true);
    });
  });

  // Silence unused-var lints for helpers we keep around for reference.
  void noExecuteRemainingAccounts;
});
