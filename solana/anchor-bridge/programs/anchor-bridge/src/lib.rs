use anchor_lang::{prelude::*, solana_program};

mod api;
mod types;

use types::{Index, BridgeConfig, Chain};

// Without this, Anchor's derivation macros break. It requires names with no path components at all
// otherwise it errors.
use anchor_bridge::Bridge;

/// chain id of this chain
pub const CHAIN_ID_SOLANA: u8 = Chain::Solana as u8;
/// maximum number of guardians
pub const MAX_LEN_GUARDIAN_KEYS: usize = 20;

#[derive(Accounts)]
pub struct VerifySig<'info> {
    pub system: AccountInfo<'info>,
    pub instruction_sysvar: AccountInfo<'info>,
    pub bridge_info: ProgramState<'info, BridgeInfo>,
    pub sig_info: AccountInfo<'info>,
    pub guardian_set_info: ProgramState<'info, GuardianSetInfo>,
    pub payer_info: AccountInfo<'info>,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, Debug)]
pub struct VerifySigsData {
    /// hash of the VAA
    pub hash: [u8; 32],
    /// instruction indices of signers (-1 for missing)
    pub signers: [i8; MAX_LEN_GUARDIAN_KEYS],
    /// indicates whether this verification should only succeed if the sig account does not exist
    pub initial_creation: bool,
}


#[derive(Accounts)]
pub struct Initialize<'info> {
    /// Account used to pay for auxillary instructions.
    #[account(signer)]
    pub payer: AccountInfo<'info>,

    /// Information about the current guardian set.
    #[account(init, associated = state)]
    pub guardian_set: ProgramAccount<'info, GuardianSetInfo>,

    /// State struct, derived by #[state], used for associated accounts.
    pub state: ProgramState<'info, Bridge>,

    /// Used for timestamping actions.
    pub clock: Sysvar<'info, Clock>,

    /// Required by Anchor for associated accounts.
    pub rent: Sysvar<'info, Rent>,

    /// Required by Anchor for associated accounts.
    pub system_program: AccountInfo<'info>,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, Debug)]
pub struct InitializeData {
    /// number of initial guardians
    pub len_guardians: u8,
    /// guardians that are allowed to sign mints
    pub initial_guardian_keys: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    /// config for the bridge
    pub config: BridgeConfig,
}

#[derive(Accounts)]
pub struct PublishMessage<'info> {
    /// No need to verify - only used as the fee payer for account creation.
    #[account(signer)]
    pub payer: AccountInfo<'info>,

    /// The emitter, only used as metadata. We verify that the account is a signer to prevent
    /// messages from being spoofed.
    #[account(signer)]
    pub emitter: AccountInfo<'info>,

    /// The message account to store data in, note that this cannot be derived by serum and so the
    /// pulish_message handler does this by hand.
    pub message: AccountInfo<'info>,

    /// State struct, derived by #[state], used for associated accounts.
    pub state: ProgramState<'info, Bridge>,

    /// Instructions used for transaction reflection.
    pub instructions: AccountInfo<'info>,

    /// Clock used for timestamping.
    pub clock: Sysvar<'info, Clock>,

    /// Required by Anchor for associated accounts.
    pub rent: Sysvar<'info, Rent>,

    /// Required by Anchor for associated accounts.
    pub system_program: AccountInfo<'info>,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, Debug)]
pub struct PublishMessageData {}

#[program]
pub mod anchor_bridge {
    use super::*;

    #[state]
    pub struct Bridge {
        pub guardian_set_index: types::Index,
        pub config: types::BridgeConfig,
    }

    /// Trick Anchor into generating Initialize client structs. Anchor generates a Pubkey only
    /// version of every Context struct, but only if a function or method with a self parameter
    /// uses it. Bridge::new does not get picked up.
    pub fn __trick_anchor_initialize(ctx: Context<Initialize>) -> Result<()> {
        Ok(())
    }

    impl Bridge {
        pub fn new(ctx: Context<Initialize>, data: InitializeData) -> Result<Self> {
            api::initialize(
                ctx,
                data.len_guardians,
                data.initial_guardian_keys,
                data.config,
            )
        }

        pub fn publish_message(&mut self, ctx: Context<PublishMessage>, data: PublishMessageData, nonce: u8) -> Result<()> {
            // Sysvar trait not implemented for Instructions by sdk, so manual check required.  See
            // the VerifySig struct for more info.
            if *ctx.accounts.instructions.key != solana_program::sysvar::instructions::id() {
                return Err(ErrorCode::InvalidSysVar.into());
            }

            api::publish_message(
                self,
                ctx,
                nonce,
            )
        }

        pub fn verify_signatures(&mut self, ctx: Context<VerifySig>, data: VerifySigsData) -> Result<()> {
            // Sysvar trait not implemented for Instructions by sdk, so manual check required.  See
            // the VerifySig struct for more info.
            if *ctx.accounts.instruction_sysvar.key != solana_program::sysvar::instructions::id() {
                return Err(ErrorCode::InvalidSysVar.into());
            }

            api::verify_signatures(
                self,
                ctx,
                data.hash,
                data.signers,
                data.initial_creation,
            )
        }
    }
}

#[error]
pub enum ErrorCode {
    #[msg("System account pubkey did not match expected address.")]
    InvalidSysVar,
}

#[account]
pub struct BridgeInfo {}

#[associated]
pub struct GuardianSetInfo {
    /// Version number of this guardian set.
    pub index: Index,
    /// Number of keys stored
    pub len_keys: u8,
    /// public key hashes of the guardian set
    pub keys: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    /// creation time
    pub creation_time: u32,
    /// expiration time when VAAs issued by this set are no longer valid
    pub expiration_time: u32,
}

/// Record of a posted wormhole message.
#[account]
#[derive(Default)]
pub struct PostedMessage {
    /// header of the posted VAA
    pub vaa_version: u8,

    /// time the vaa was submitted
    pub vaa_time: u32,

    /// Account where signatures are stored
    pub vaa_signature_account: Pubkey,

    /// time the posted message was created
    pub submission_time: u32,

    /// unique nonce for this message
    pub nonce: u32,

    /// emitter of the message
    pub emitter_chain: Chain,

    /// emitter of the message
    pub emitter_address: [u8; 32],

    /// message payload
    pub payload: [[u8; 32]; 13],
}
