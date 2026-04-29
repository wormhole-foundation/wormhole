declare_program!(wormhole_verify_vaa_shim);
use anchor_lang::{
    prelude::*,
    solana_program::{
        self,
        instruction::{AccountMeta, Instruction},
        keccak,
        program::invoke_signed,
    },
};
use wormhole_raw_vaas::Body;
use wormhole_verify_vaa_shim::cpi::accounts::VerifyHash;
use wormhole_verify_vaa_shim::program::WormholeVerifyVaaShim;

declare_id!("WPSrHuFyxkpQkz67yQKCQf5qHCDMfnoA6gpAacvB1Hn");

pub const WORMHOLE_PAUSER_VERSION: &str = "WormholePauser-0.0.1";

pub const GOVERNANCE_CHAIN_ID: u16 = 1;
pub const GOVERNANCE_EMITTER: [u8; 32] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
];

/// "DelegatedPauser" left-padded to 32 bytes. See whitepapers/0018_pauser.md.
pub const DELEGATED_PAUSER_MODULE: [u8; 32] = [
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x44, 0x65, 0x6C, 0x65, 0x67, 0x61, 0x74, 0x65, 0x64, 0x50, 0x61, 0x75, 0x73, 0x65, 0x72,
];

/// Action 2: SetConfigSolana (32-byte signer pubkeys). Action 1 (SetConfigEvm) is rejected here.
pub const ACTION_SET_CONFIG_SOLANA: u8 = 2;

/// Wormhole chain ID for Solana.
pub const OUR_CHAIN: u16 = 1;

/// Hard cap: numSigners is wire-encoded as a u8.
pub const MAX_SIGNERS: usize = u8::MAX as usize;

/// Bitmap covering up to 256 signers (255 indices needed, rounded up to whole bytes).
pub const APPROVAL_BITMAP_BYTES: usize = 32;

// Governance payload (after the 51-byte VAA body header):
//   module(32) + action(1) + chain(2) + index(2) + threshold(1) + expiry(8) + numSigners(1) + signers(numSigners*32)
const MIN_GOVERNANCE_PAYLOAD_SIZE: usize = 32 + 1 + 2 + 2 + 1 + 8 + 1;

/// Zero-copy parser for a DelegatedPauser SetConfigSolana governance payload.
#[derive(Clone, Copy)]
pub struct SetConfigPayload<'a> {
    span: &'a [u8],
}

impl<'a> SetConfigPayload<'a> {
    pub fn parse(span: &'a [u8]) -> Result<Self> {
        require!(
            span.len() >= MIN_GOVERNANCE_PAYLOAD_SIZE,
            WormholePauserError::GovernancePayloadTooShort
        );
        Ok(Self { span })
    }

    pub fn module(&self) -> [u8; 32] {
        let mut out = [0u8; 32];
        out.copy_from_slice(&self.span[0..32]);
        out
    }

    pub fn action(&self) -> u8 {
        self.span[32]
    }

    pub fn target_chain(&self) -> u16 {
        u16::from_be_bytes([self.span[33], self.span[34]])
    }

    pub fn index(&self) -> u16 {
        u16::from_be_bytes([self.span[35], self.span[36]])
    }

    pub fn threshold(&self) -> u8 {
        self.span[37]
    }

    pub fn expiry_duration(&self) -> u64 {
        u64::from_be_bytes([
            self.span[38],
            self.span[39],
            self.span[40],
            self.span[41],
            self.span[42],
            self.span[43],
            self.span[44],
            self.span[45],
        ])
    }

    pub fn num_signers(&self) -> u8 {
        self.span[46]
    }

    /// Tail bytes after the fixed body, expected to be `num_signers * 32`.
    pub fn signer_bytes(&self) -> &'a [u8] {
        &self.span[47..]
    }
}

#[program]
pub mod wormhole_pauser {
    use super::*;

    /// Apply a `SetConfigSolana` governance VAA (DelegatedPauser module, action 2).
    /// Replaces the on-chain signer set, threshold, and expiry duration. The new index
    /// in the message must be exactly `currentIndex + 1`.
    pub fn submit_config(ctx: Context<SubmitConfig>, args: SubmitConfigArgs) -> Result<()> {
        // Confirm args.digest is the double-keccak of args.vaa_body so the seed-derived
        // `consumed` PDA is bound to the actual VAA body the instruction parses.
        let message_hash = solana_program::keccak::hashv(&[&args.vaa_body]).to_bytes();
        let digest = keccak::hash(message_hash.as_slice()).to_bytes();
        require!(args.digest == digest, WormholePauserError::DigestMismatch);

        // Verify the VAA hash against the guardian signatures via the verify-vaa shim.
        wormhole_verify_vaa_shim::cpi::verify_hash(
            CpiContext::new(
                ctx.accounts.wormhole_verify_vaa_shim.to_account_info(),
                VerifyHash {
                    guardian_set: ctx.accounts.guardian_set.to_account_info(),
                    guardian_signatures: ctx.accounts.guardian_signatures.to_account_info(),
                },
            ),
            args.guardian_set_bump,
            digest,
        )?;

        let body =
            Body::parse(&args.vaa_body).map_err(|_| WormholePauserError::InvalidVaaBody)?;
        require!(
            body.emitter_chain() == GOVERNANCE_CHAIN_ID,
            WormholePauserError::InvalidGovernanceChain
        );
        require!(
            body.emitter_address() == GOVERNANCE_EMITTER,
            WormholePauserError::InvalidGovernanceEmitter
        );

        let payload = body.payload();
        let parsed = SetConfigPayload::parse(payload.as_ref())?;

        require!(
            parsed.module() == DELEGATED_PAUSER_MODULE,
            WormholePauserError::InvalidGovernanceModule
        );
        require!(
            parsed.action() == ACTION_SET_CONFIG_SOLANA,
            WormholePauserError::InvalidGovernanceAction
        );
        require!(
            parsed.target_chain() == OUR_CHAIN,
            WormholePauserError::InvalidTargetChain
        );

        let config = &mut ctx.accounts.config;

        let new_index = parsed.index();
        require!(
            new_index == config.config_index.checked_add(1).unwrap(),
            WormholePauserError::InvalidIndex
        );

        let new_threshold = parsed.threshold();
        require!(new_threshold > 0, WormholePauserError::InvalidThreshold);

        let new_expiry = parsed.expiry_duration();
        require!(new_expiry > 0, WormholePauserError::InvalidExpiryDuration);

        let num_signers = parsed.num_signers();
        require!(num_signers > 0, WormholePauserError::EmptySignerSet);
        require!(
            new_threshold <= num_signers,
            WormholePauserError::InvalidThreshold
        );

        // The wire format requires `numSigners * 32` trailing bytes. Reject any other length.
        let signer_bytes = parsed.signer_bytes();
        require!(
            signer_bytes.len() == (num_signers as usize) * 32,
            WormholePauserError::InvalidPayloadLength
        );

        let mut new_signers: Vec<Pubkey> = Vec::with_capacity(num_signers as usize);
        for i in 0..num_signers as usize {
            let mut buf = [0u8; 32];
            buf.copy_from_slice(&signer_bytes[i * 32..(i + 1) * 32]);
            let pk = Pubkey::new_from_array(buf);
            require!(pk != Pubkey::default(), WormholePauserError::ZeroSigner);
            require!(
                !new_signers.contains(&pk),
                WormholePauserError::DuplicateSigner
            );
            new_signers.push(pk);
        }

        config.config_index = new_index;
        config.threshold = new_threshold;
        config.expiry_duration = new_expiry;
        config.signers = new_signers.clone();

        emit!(ConfigSet {
            config_index: new_index,
            threshold: new_threshold,
            expiry_duration: new_expiry,
            signers: new_signers,
        });

        Ok(())
    }

    /// Create a new pause proposal. The caller must be in the current signer set; they are
    /// auto-approved, so a `threshold == 1` config executes the CPI in this same transaction.
    /// Remaining accounts (when execution may occur) must be `[target_program, ...accounts]`
    /// with `accounts` covering every entry in `args.account_metas`.
    pub fn propose<'c: 'info, 'info>(
        ctx: Context<'_, '_, 'c, 'info, Propose<'info>>,
        args: ProposeArgs,
    ) -> Result<()> {
        require!(
            args.account_metas.len() <= u8::MAX as usize,
            WormholePauserError::TooManyAccounts
        );

        let signer_index = find_signer_index(&ctx.accounts.config.signers, &ctx.accounts.signer.key())?;

        let proposal_id = ctx.accounts.config.next_proposal_id;
        let config_index = ctx.accounts.config.config_index;
        let expiry_duration = ctx.accounts.config.expiry_duration;
        let threshold = ctx.accounts.config.threshold;

        require!(config_index > 0, WormholePauserError::ConfigNotInitialized);

        let now = Clock::get()?.unix_timestamp as u64;
        let expires_at = now
            .checked_add(expiry_duration)
            .ok_or(WormholePauserError::ExpiryOverflow)?;

        let proposal = &mut ctx.accounts.proposal;
        proposal.executed = false;
        proposal.config_index = config_index;
        proposal.expires_at = expires_at;
        proposal.approval_count = 0;
        proposal.approvals = [0u8; APPROVAL_BITMAP_BYTES];
        proposal.target_program = args.target_program;
        proposal.account_metas = args.account_metas;
        proposal.data = args.data;

        emit!(ProposalProposed {
            proposal_id,
            proposer: ctx.accounts.signer.key(),
            target_program: proposal.target_program,
            config_index,
            expires_at,
        });

        // Reuse the same record-and-maybe-execute path used by `approve` so that a
        // `threshold == 1` config proposes-and-executes atomically.
        record_approval_and_maybe_execute(
            proposal,
            &ctx.accounts.signer.key(),
            signer_index,
            threshold,
            proposal_id,
            &ctx.accounts.authority,
            ctx.bumps.authority,
            ctx.remaining_accounts,
        )?;

        ctx.accounts.config.next_proposal_id = proposal_id
            .checked_add(1)
            .ok_or(WormholePauserError::ProposalIdOverflow)?;

        Ok(())
    }

    /// Approve an existing proposal. If this approval meets the threshold, the proposal's
    /// CPI is executed in the same transaction; if it reverts, the entire tx reverts.
    /// Remaining accounts on a threshold-meeting call must be `[target_program, ...accounts]`.
    pub fn approve<'c: 'info, 'info>(
        ctx: Context<'_, '_, 'c, 'info, Approve<'info>>,
        args: ApproveArgs,
    ) -> Result<()> {
        let signer_index =
            find_signer_index(&ctx.accounts.config.signers, &ctx.accounts.signer.key())?;
        let threshold = ctx.accounts.config.threshold;
        let current_config_index = ctx.accounts.config.config_index;

        let proposal = &mut ctx.accounts.proposal;
        require_active(proposal, current_config_index)?;
        require!(
            !is_bit_set(&proposal.approvals, signer_index),
            WormholePauserError::AlreadyApproved
        );

        record_approval_and_maybe_execute(
            proposal,
            &ctx.accounts.signer.key(),
            signer_index,
            threshold,
            args.proposal_id,
            &ctx.accounts.authority,
            ctx.bumps.authority,
            ctx.remaining_accounts,
        )?;

        Ok(())
    }

    /// Cancel the caller's prior approval of a still-active proposal.
    pub fn cancel_approval(ctx: Context<CancelApproval>, args: CancelApprovalArgs) -> Result<()> {
        let signer_index =
            find_signer_index(&ctx.accounts.config.signers, &ctx.accounts.signer.key())?;
        let current_config_index = ctx.accounts.config.config_index;

        let proposal = &mut ctx.accounts.proposal;
        require_active(proposal, current_config_index)?;
        require!(
            is_bit_set(&proposal.approvals, signer_index),
            WormholePauserError::NotApproved
        );

        clear_bit(&mut proposal.approvals, signer_index);
        proposal.approval_count = proposal
            .approval_count
            .checked_sub(1)
            .ok_or(WormholePauserError::ApprovalUnderflow)?;

        emit!(ProposalApprovalCancelled {
            proposal_id: args.proposal_id,
            signer: ctx.accounts.signer.key(),
            approval_count: proposal.approval_count,
        });

        Ok(())
    }
}

fn find_signer_index(signers: &[Pubkey], who: &Pubkey) -> Result<u8> {
    signers
        .iter()
        .position(|s| s == who)
        .map(|i| i as u8)
        .ok_or_else(|| WormholePauserError::NotSigner.into())
}

fn require_active(proposal: &Proposal, current_config_index: u16) -> Result<()> {
    require!(
        !proposal.executed,
        WormholePauserError::ProposalAlreadyExecuted
    );
    let now = Clock::get()?.unix_timestamp as u64;
    require!(now < proposal.expires_at, WormholePauserError::ProposalExpired);
    require!(
        proposal.config_index == current_config_index,
        WormholePauserError::ProposalConfigRotated
    );
    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn record_approval_and_maybe_execute<'info>(
    proposal: &mut Account<'info, Proposal>,
    signer_key: &Pubkey,
    signer_index: u8,
    threshold: u8,
    proposal_id: u64,
    authority: &UncheckedAccount<'info>,
    authority_bump: u8,
    remaining_accounts: &[AccountInfo<'info>],
) -> Result<()> {
    set_bit(&mut proposal.approvals, signer_index);
    proposal.approval_count = proposal
        .approval_count
        .checked_add(1)
        .ok_or(WormholePauserError::ApprovalOverflow)?;
    let approval_count = proposal.approval_count;

    emit!(ProposalApproved {
        proposal_id,
        signer: *signer_key,
        approval_count,
    });

    if approval_count >= threshold {
        // Effect first, then interaction. If the CPI reverts, the whole tx reverts (along with
        // this approval), so signers can retry without permanently consuming their approval.
        proposal.executed = true;
        execute_proposal_cpi(
            proposal.target_program,
            &proposal.account_metas,
            &proposal.data,
            authority,
            authority_bump,
            remaining_accounts,
        )?;
        emit!(ProposalExecuted { proposal_id });
    }
    Ok(())
}

fn execute_proposal_cpi<'info>(
    target_program: Pubkey,
    account_metas: &[ProposalAccountMeta],
    data: &[u8],
    authority: &UncheckedAccount<'info>,
    authority_bump: u8,
    remaining_accounts: &[AccountInfo<'info>],
) -> Result<()> {
    require!(
        !remaining_accounts.is_empty(),
        WormholePauserError::MissingTargetProgram
    );
    let target_program_info = &remaining_accounts[0];
    require!(
        *target_program_info.key == target_program,
        WormholePauserError::TargetProgramMismatch
    );

    let metas: Vec<AccountMeta> = account_metas
        .iter()
        .map(|m| {
            if m.is_writable {
                AccountMeta::new(m.pubkey, m.is_signer)
            } else {
                AccountMeta::new_readonly(m.pubkey, m.is_signer)
            }
        })
        .collect();

    let ix = Instruction {
        program_id: target_program,
        accounts: metas,
        data: data.to_vec(),
    };

    // The runtime will resolve every referenced AccountInfo from the slice we pass it.
    // Include the authority (so its is_signer flag can be satisfied via seeds) plus everything
    // the caller forwarded as remaining_accounts.
    let mut infos: Vec<AccountInfo<'info>> = Vec::with_capacity(remaining_accounts.len() + 1);
    infos.push(authority.to_account_info());
    for acc in remaining_accounts.iter() {
        infos.push(acc.clone());
    }

    let signer_seeds: &[&[&[u8]]] = &[&[AUTHORITY_SEED, &[authority_bump]]];
    invoke_signed(&ix, &infos, signer_seeds)
        .map_err(|_| WormholePauserError::ExecutionFailed.into())
        .map(|_| ())
}

fn set_bit(bitmap: &mut [u8; APPROVAL_BITMAP_BYTES], index: u8) {
    let i = index as usize;
    bitmap[i / 8] |= 1u8 << (i % 8);
}

fn clear_bit(bitmap: &mut [u8; APPROVAL_BITMAP_BYTES], index: u8) {
    let i = index as usize;
    bitmap[i / 8] &= !(1u8 << (i % 8));
}

fn is_bit_set(bitmap: &[u8; APPROVAL_BITMAP_BYTES], index: u8) -> bool {
    let i = index as usize;
    (bitmap[i / 8] >> (i % 8)) & 1 == 1
}

#[derive(AnchorSerialize, AnchorDeserialize)]
pub struct SubmitConfigArgs {
    pub guardian_set_bump: u8,
    pub digest: [u8; 32],
    pub vaa_body: Vec<u8>,
}

#[derive(AnchorSerialize, AnchorDeserialize)]
pub struct ProposeArgs {
    pub target_program: Pubkey,
    pub account_metas: Vec<ProposalAccountMeta>,
    pub data: Vec<u8>,
}

#[derive(AnchorSerialize, AnchorDeserialize)]
pub struct ApproveArgs {
    pub proposal_id: u64,
}

#[derive(AnchorSerialize, AnchorDeserialize)]
pub struct CancelApprovalArgs {
    pub proposal_id: u64,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, Debug, PartialEq, Eq)]
pub struct ProposalAccountMeta {
    pub pubkey: Pubkey,
    pub is_signer: bool,
    pub is_writable: bool,
}

impl ProposalAccountMeta {
    pub const SERIALIZED_SIZE: usize = 32 + 1 + 1;
}

pub const AUTHORITY_SEED: &[u8] = b"authority";

/// Singleton config PDA. Holds the current signer set, threshold, expiry duration,
/// monotonic config index, and next proposal id.
#[account]
pub struct Config {
    pub config_index: u16,
    pub threshold: u8,
    pub expiry_duration: u64,
    pub next_proposal_id: u64,
    pub signers: Vec<Pubkey>,
}

impl Config {
    pub const SEED_PREFIX: &'static [u8] = b"config";

    /// Maximum size, sized to accommodate `MAX_SIGNERS` so the account is allocated once
    /// at first init and reused across config rotations without realloc.
    pub const MAX_SIZE: usize = 8 // discriminator
        + 2 // config_index
        + 1 // threshold
        + 8 // expiry_duration
        + 8 // next_proposal_id
        + 4 + (32 * MAX_SIGNERS); // signers vec
}

/// Per-proposal PDA. Created by `propose`, never closed (signers can read past results).
#[account]
pub struct Proposal {
    pub executed: bool,
    pub config_index: u16,
    pub expires_at: u64,
    pub approval_count: u8,
    pub approvals: [u8; APPROVAL_BITMAP_BYTES],
    pub target_program: Pubkey,
    pub account_metas: Vec<ProposalAccountMeta>,
    pub data: Vec<u8>,
}

impl Proposal {
    pub const SEED_PREFIX: &'static [u8] = b"proposal";

    pub fn size(num_account_metas: usize, data_len: usize) -> usize {
        8 // discriminator
        + 1 // executed
        + 2 // config_index
        + 8 // expires_at
        + 1 // approval_count
        + APPROVAL_BITMAP_BYTES // approvals
        + 32 // target_program
        + 4 + (ProposalAccountMeta::SERIALIZED_SIZE * num_account_metas) // account_metas vec
        + 4 + data_len // data vec
    }
}

#[derive(Accounts)]
#[instruction(args: SubmitConfigArgs)]
pub struct SubmitConfig<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,

    /// CHECK: Guardian set used for signature verification by the shim.
    pub guardian_set: UncheckedAccount<'info>,

    /// CHECK: Stored guardian signatures, ownership and discriminator checked by the shim.
    pub guardian_signatures: UncheckedAccount<'info>,

    /// CHECK: Replay-protection PDA. Created on success; subsequent submits with the same
    /// digest fail to allocate.
    #[account(
        init,
        payer = payer,
        space = 0,
        seeds = [b"consumed_vaa", args.digest.as_ref()],
        bump,
    )]
    pub consumed: UncheckedAccount<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        space = Config::MAX_SIZE,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    pub config: Account<'info, Config>,

    pub wormhole_verify_vaa_shim: Program<'info, WormholeVerifyVaaShim>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(args: ProposeArgs)]
pub struct Propose<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,

    /// Caller must be present in the current signer set. Verified in the instruction body.
    pub signer: Signer<'info>,

    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    pub config: Account<'info, Config>,

    #[account(
        init,
        payer = payer,
        space = Proposal::size(args.account_metas.len(), args.data.len()),
        seeds = [Proposal::SEED_PREFIX, &config.next_proposal_id.to_le_bytes()],
        bump,
    )]
    pub proposal: Account<'info, Proposal>,

    /// CHECK: PDA used to sign downstream CPIs (`invoke_signed` with the AUTHORITY_SEED bump).
    /// Verified by the seeds constraint.
    #[account(
        seeds = [AUTHORITY_SEED],
        bump,
    )]
    pub authority: UncheckedAccount<'info>,

    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(args: ApproveArgs)]
pub struct Approve<'info> {
    pub signer: Signer<'info>,

    #[account(
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    pub config: Account<'info, Config>,

    #[account(
        mut,
        seeds = [Proposal::SEED_PREFIX, &args.proposal_id.to_le_bytes()],
        bump,
    )]
    pub proposal: Account<'info, Proposal>,

    /// CHECK: PDA used to sign downstream CPIs.
    #[account(
        seeds = [AUTHORITY_SEED],
        bump,
    )]
    pub authority: UncheckedAccount<'info>,
}

#[derive(Accounts)]
#[instruction(args: CancelApprovalArgs)]
pub struct CancelApproval<'info> {
    pub signer: Signer<'info>,

    #[account(
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    pub config: Account<'info, Config>,

    #[account(
        mut,
        seeds = [Proposal::SEED_PREFIX, &args.proposal_id.to_le_bytes()],
        bump,
    )]
    pub proposal: Account<'info, Proposal>,
}

#[event]
pub struct ConfigSet {
    pub config_index: u16,
    pub threshold: u8,
    pub expiry_duration: u64,
    pub signers: Vec<Pubkey>,
}

#[event]
pub struct ProposalProposed {
    pub proposal_id: u64,
    pub proposer: Pubkey,
    pub target_program: Pubkey,
    pub config_index: u16,
    pub expires_at: u64,
}

#[event]
pub struct ProposalApproved {
    pub proposal_id: u64,
    pub signer: Pubkey,
    pub approval_count: u8,
}

#[event]
pub struct ProposalApprovalCancelled {
    pub proposal_id: u64,
    pub signer: Pubkey,
    pub approval_count: u8,
}

#[event]
pub struct ProposalExecuted {
    pub proposal_id: u64,
}

#[error_code]
pub enum WormholePauserError {
    #[msg("Digest argument does not match computed digest from VAA body")]
    DigestMismatch,
    #[msg("Failed to parse VAA body")]
    InvalidVaaBody,
    #[msg("VAA is not from the governance chain")]
    InvalidGovernanceChain,
    #[msg("VAA is not from the governance emitter")]
    InvalidGovernanceEmitter,
    #[msg("Governance payload too short")]
    GovernancePayloadTooShort,
    #[msg("Invalid governance module")]
    InvalidGovernanceModule,
    #[msg("Invalid governance action for this platform")]
    InvalidGovernanceAction,
    #[msg("Governance message target chain does not match this chain")]
    InvalidTargetChain,
    #[msg("Governance config index must be exactly currentIndex + 1")]
    InvalidIndex,
    #[msg("Threshold must be > 0 and <= numSigners")]
    InvalidThreshold,
    #[msg("Expiry duration must be > 0")]
    InvalidExpiryDuration,
    #[msg("Signer set must be non-empty")]
    EmptySignerSet,
    #[msg("Signer cannot be the all-zero pubkey")]
    ZeroSigner,
    #[msg("Signer set contains a duplicate")]
    DuplicateSigner,
    #[msg("Governance payload length does not match numSigners")]
    InvalidPayloadLength,
    #[msg("Caller is not in the current signer set")]
    NotSigner,
    #[msg("Config has not been initialized via governance yet")]
    ConfigNotInitialized,
    #[msg("Proposal expiry overflowed u64")]
    ExpiryOverflow,
    #[msg("Next proposal ID overflowed u64")]
    ProposalIdOverflow,
    #[msg("Approval count overflowed u8")]
    ApprovalOverflow,
    #[msg("Approval count underflowed u8")]
    ApprovalUnderflow,
    #[msg("Too many account metas (max 255)")]
    TooManyAccounts,
    #[msg("Proposal has already been executed")]
    ProposalAlreadyExecuted,
    #[msg("Proposal has expired")]
    ProposalExpired,
    #[msg("Proposal config index does not match current config (config rotated)")]
    ProposalConfigRotated,
    #[msg("Caller has already approved this proposal")]
    AlreadyApproved,
    #[msg("Caller has not approved this proposal")]
    NotApproved,
    #[msg("Remaining accounts must include the target program at index 0")]
    MissingTargetProgram,
    #[msg("Provided target program does not match proposal.target_program")]
    TargetProgramMismatch,
    #[msg("Target instruction CPI failed")]
    ExecutionFailed,
}
