declare_program!(wormhole_verify_vaa_shim);
use anchor_lang::{
    prelude::*,
    solana_program::{self, keccak},
};
use wormhole_raw_vaas::Body;
use wormhole_verify_vaa_shim::cpi::accounts::VerifyHash;
use wormhole_verify_vaa_shim::program::WormholeVerifyVaaShim;

declare_id!("wdmsTJP6YnsfeQjPuuEzGCrHmZvTmNy8VkxMCK8JkBX");

pub const GOVERNANCE_CHAIN_ID: u16 = 1;
pub const GOVERNANCE_EMITTER: [u8; 32] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
];

/// DelegatedManagerModule is the 32-byte module identifier for delegated manager governance.
/// It is "DelegatedManager" left-padded with zeros.
pub const DELEGATED_MANAGER_MODULE: [u8; 32] = [
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x44, 0x65, 0x6C, 0x65, 0x67, 0x61, 0x74, 0x65, 0x64, 0x4D, 0x61, 0x6E, 0x61, 0x67, 0x65, 0x72,
];

/// ActionManagerSetUpdate is the governance action for updating a manager set.
pub const ACTION_MANAGER_SET_UPDATE: u8 = 1;

/// The Wormhole chain ID for this chain (Solana).
static OUR_CHAIN: u16 = 1;

/// Minimum payload size: module (32) + action (1) + target_chain (2) + manager_chain (2) + manager_set_index (4) = 41
const MIN_GOVERNANCE_PAYLOAD_SIZE: usize = 41;

// VAA body offsets (for zero-copy extraction in account constraints)
// VAA body: timestamp(4) + nonce(4) + emitter_chain(2) + emitter_address(32) + sequence(8) + consistency_level(1) = 51 bytes header
// Governance payload: module(32) + action(1) + target_chain(2) + manager_chain_id(2) + manager_set_index(4) + manager_set(variable)
// Absolute offsets from vaa_body start:
const MANAGER_CHAIN_ID_OFFSET: usize = 86; // 51 + 35
const MANAGER_SET_INDEX_OFFSET: usize = 88; // 51 + 37
const MANAGER_SET_DATA_OFFSET: usize = 92; // 51 + 41

/// Zero-copy parser for DelegatedManager governance payloads.
#[derive(Clone, Copy)]
pub struct ManagerSetUpdatePayload<'a> {
    span: &'a [u8],
}

impl<'a> ManagerSetUpdatePayload<'a> {
    /// Parse a governance payload for a manager set update.
    pub fn parse(span: &'a [u8]) -> Result<Self> {
        require!(
            span.len() >= MIN_GOVERNANCE_PAYLOAD_SIZE,
            DelegatedManagerSetError::GovernancePayloadTooShort
        );
        Ok(Self { span })
    }

    /// Returns the 32-byte module identifier.
    pub fn module(&self) -> [u8; 32] {
        let mut module = [0u8; 32];
        module.copy_from_slice(&self.span[0..32]);
        module
    }

    /// Returns the governance action (1 byte).
    pub fn action(&self) -> u8 {
        self.span[32]
    }

    /// Returns the target chain ID (2 bytes, big-endian). 0 = universal.
    pub fn target_chain(&self) -> u16 {
        u16::from_be_bytes([self.span[33], self.span[34]])
    }

    /// Returns the manager chain ID (2 bytes, big-endian).
    pub fn manager_chain_id(&self) -> u16 {
        u16::from_be_bytes([self.span[35], self.span[36]])
    }

    /// Returns the new manager set index (4 bytes, big-endian).
    pub fn new_manager_set_index(&self) -> u32 {
        u32::from_be_bytes([self.span[37], self.span[38], self.span[39], self.span[40]])
    }

    /// Returns the new manager set bytes (remaining bytes after header).
    pub fn new_manager_set(&self) -> &'a [u8] {
        &self.span[41..]
    }
}

#[program]
pub mod delegated_manager_set {
    use super::*;

    pub fn submit_new_manager_set(
        ctx: Context<SubmitNewManagerSet>,
        args: SubmitNewManagerSetArgs,
    ) -> Result<()> {
        // Compute the message hash.
        let message_hash = &solana_program::keccak::hashv(&[&args.vaa_body]).to_bytes();
        let digest = keccak::hash(message_hash.as_slice()).to_bytes();

        // Confirm the digest argument matches the computed one
        require!(
            args.digest == digest,
            DelegatedManagerSetError::DigestMismatch
        );

        // Verify the hash against the signatures.
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

        // Decode the VAA body
        let body =
            Body::parse(&args.vaa_body).map_err(|_| DelegatedManagerSetError::InvalidVaaBody)?;

        // Verify the VAA is from the governance emitter
        require!(
            body.emitter_chain() == GOVERNANCE_CHAIN_ID,
            DelegatedManagerSetError::InvalidGovernanceChain
        );
        require!(
            body.emitter_address() == GOVERNANCE_EMITTER,
            DelegatedManagerSetError::InvalidGovernanceEmitter
        );

        // Parse the governance payload
        let payload = body.payload();
        let governance_payload = ManagerSetUpdatePayload::parse(payload.as_ref())?;

        // Validate the governance module
        require!(
            governance_payload.module() == DELEGATED_MANAGER_MODULE,
            DelegatedManagerSetError::InvalidGovernanceModule
        );

        // Validate the governance action
        require!(
            governance_payload.action() == ACTION_MANAGER_SET_UPDATE,
            DelegatedManagerSetError::InvalidGovernanceAction
        );

        // Target chain must be 0 (universal) or match our chain
        let target_chain = governance_payload.target_chain();
        require!(
            target_chain == 0 || target_chain == OUR_CHAIN,
            DelegatedManagerSetError::InvalidTargetChain
        );

        let manager_chain_id = governance_payload.manager_chain_id();
        let new_manager_set_index = governance_payload.new_manager_set_index();
        let new_manager_set = governance_payload.new_manager_set();

        // Validate the manager set index is incrementing by 1
        let current_index = ctx.accounts.manager_set_index.current_index;
        require!(
            new_manager_set_index == current_index + 1,
            DelegatedManagerSetError::InvalidManagerSetIndex
        );

        // Store the new manager set
        let manager_set_account = &mut ctx.accounts.manager_set;
        manager_set_account.manager_chain_id = manager_chain_id;
        manager_set_account.index = new_manager_set_index;
        manager_set_account.manager_set = new_manager_set.to_vec();

        // Update the current manager set index
        let manager_set_index_account = &mut ctx.accounts.manager_set_index;
        manager_set_index_account.manager_chain_id = manager_chain_id;
        manager_set_index_account.current_index = new_manager_set_index;

        Ok(())
    }
}

#[derive(AnchorSerialize, AnchorDeserialize)]
pub struct SubmitNewManagerSetArgs {
    pub guardian_set_bump: u8,
    pub digest: [u8; 32],
    pub vaa_body: Vec<u8>,
}

/// Stores the current manager set index for a given chain ID.
/// PDA seeds: ["manager_set_index", manager_chain_id]
#[account]
pub struct ManagerSetIndex {
    /// The manager chain ID this index is for.
    pub manager_chain_id: u16,
    /// The current manager set index.
    pub current_index: u32,
}

impl ManagerSetIndex {
    pub const SEED_PREFIX: &'static [u8] = b"manager_set_index";

    pub const SIZE: usize = 8 + // discriminator
        2 + // manager_chain_id
        4; // current_index
}

/// Stores a manager set for a given chain ID and index.
/// PDA seeds: ["manager_set", manager_chain_id, manager_set_index]
#[account]
pub struct ManagerSet {
    /// The manager chain ID this set is for.
    pub manager_chain_id: u16,
    /// The manager set index.
    pub index: u32,
    /// The raw manager set bytes.
    pub manager_set: Vec<u8>,
}

impl ManagerSet {
    pub const SEED_PREFIX: &'static [u8] = b"manager_set";

    pub fn size(manager_set_len: usize) -> usize {
        8 + // discriminator
        2 + // manager_chain_id
        4 + // index
        4 + manager_set_len // manager_set (vec with length prefix)
    }
}

#[derive(Accounts)]
#[instruction(args: SubmitNewManagerSetArgs)]
pub struct SubmitNewManagerSet<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,

    /// CHECK: Guardian set used for signature verification by shim.
    /// Derivation is checked by the shim.
    pub guardian_set: UncheckedAccount<'info>,

    /// CHECK: Stored guardian signatures to be verified by shim.
    /// Ownership ownership and discriminator is checked by the shim.
    pub guardian_signatures: UncheckedAccount<'info>,

    /// CHECK: This account is used as a PDA to ensure uniqueness and is not read or written to.
    /// The derivation is confirmed in the instruction
    #[account(
        init,
        payer = payer,
        space = 0,
        seeds = [b"consumed_vaa", args.digest.as_ref()],
        bump,
    )]
    pub consumed: UncheckedAccount<'info>,

    /// Stores the current manager set index for this manager chain.
    /// Initialized on first manager set update, or updated on subsequent ones.
    #[account(
        init_if_needed,
        payer = payer,
        space = ManagerSetIndex::SIZE,
        seeds = [ManagerSetIndex::SEED_PREFIX, &args.vaa_body[MANAGER_CHAIN_ID_OFFSET..MANAGER_CHAIN_ID_OFFSET + 2]],
        bump,
    )]
    pub manager_set_index: Account<'info, ManagerSetIndex>,

    /// Stores the new manager set.
    #[account(
        init,
        payer = payer,
        space = ManagerSet::size(args.vaa_body.len() - MANAGER_SET_DATA_OFFSET),
        seeds = [
            ManagerSet::SEED_PREFIX,
            &args.vaa_body[MANAGER_CHAIN_ID_OFFSET..MANAGER_CHAIN_ID_OFFSET + 2],
            &args.vaa_body[MANAGER_SET_INDEX_OFFSET..MANAGER_SET_INDEX_OFFSET + 4]
        ],
        bump,
    )]
    pub manager_set: Account<'info, ManagerSet>,

    pub wormhole_verify_vaa_shim: Program<'info, WormholeVerifyVaaShim>,

    pub system_program: Program<'info, System>,
}

#[error_code]
pub enum DelegatedManagerSetError {
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
    #[msg("Invalid governance action")]
    InvalidGovernanceAction,
    #[msg("Invalid target chain")]
    InvalidTargetChain,
    #[msg("Manager set index must increment by 1")]
    InvalidManagerSetIndex,
}
