use anchor_lang::{prelude::*, solana_program};

use crate::{
    accounts,
    anchor_bridge::Bridge,
    types::{BridgeConfig, Index, Chain},
    PublishMessage,
    PostedMessage,
    Result,
    MAX_LEN_GUARDIAN_KEYS,
};

/// Constant fee for VAA transactions, measured in lamports.
const VAA_TX_FEE: u64 = 18 * 10000;

/// Maximum size of a posted VAA
pub const MAX_PAYLOAD_SIZE: usize = 400;

pub fn publish_message(bridge: &mut Bridge, ctx: Context<PublishMessage>, nonce: u8) -> Result<()> {
    // Manually create message account, as Anchor can't do it.
    let mut message: ProgramAccount<PostedMessage> = {
        // First create the message account. 8 Bytes additional for the discriminator.
        let space = 8 + PostedMessage::default().try_to_vec().unwrap().len();
        let lamports = ctx.accounts.rent.minimum_balance(space);
        let ix = solana_program::system_instruction::create_account(
            ctx.accounts.payer.key,
            ctx.accounts.message.key,
            lamports,
            space as u64,
            ctx.program_id,
        );

        // Derived seeds for a message account.
        let seeds = [
            ctx.program_id.as_ref(),
            ctx.accounts.emitter.key.as_ref(),
            &[nonce],
        ];

        // Wrap seeds in a signer list.
        let signer = &[&seeds[..]];

        // Create account using generated data.
        solana_program::program::invoke_signed(
            &ix,
            &[
                ctx.accounts.emitter.clone(),
                ctx.accounts.system_program.clone(),
            ],
            signer,
        )?;
        // Deserialize the newly created account into an object.
        ProgramAccount::try_from_init(&ctx.accounts.message)?
    };

    // Initialize Message data.
    message.submission_time = ctx.accounts.clock.unix_timestamp as u32;
    message.emitter_chain = Chain::Solana;
    message.emitter_address = ctx.accounts.emitter.key.to_bytes();

    // Manually persist changes since we manually created the account.
    message.exit(ctx.program_id)?;

    Ok(())
}

// A const time calculation of the fee required to publish a message.
//
// Cost breakdown:
// - 2 Signatures
// - 1 Claimed VAA Rent
// - 2x Guardian Fees
const fn calculate_transfer_fee() -> u64 {
    use std::mem::size_of;
    const SIGNATURE_COST: u64 = size_of::<SignatureState>() as u64;
    const VAA_COST: u64 = size_of::<ClaimedVAA>() as u64;
    const VAA_FEE: u64 = VAA_TX_FEE;
    SIGNATURE_COST + VAA_COST + VAA_FEE
}

/// Signature state
#[repr(C)]
#[derive(Clone, Copy)]
pub struct SignatureState {
    /// signatures of validators
    pub signatures: [[u8; 65]; MAX_LEN_GUARDIAN_KEYS],

    /// hash of the data
    pub hash: [u8; 32],

    /// index of the guardian set
    pub guardian_set_index: u32,
}

/// Record of a claimed VAA
#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct ClaimedVAA {
    /// hash of the vaa
    pub hash: [u8; 32],

    /// time the vaa was submitted
    pub vaa_time: u32,
}
