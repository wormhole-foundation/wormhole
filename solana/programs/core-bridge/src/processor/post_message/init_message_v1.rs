use std::io::{Read, Write};

use crate::{
    constants::MAX_MESSAGE_PAYLOAD_SIZE,
    error::CoreBridgeError,
    legacy::utils::LegacyAccount,
    state::{MessageStatus, PostedMessageV1, PostedMessageV1Info},
    types::Commitment,
};
use anchor_lang::prelude::*;

use super::new_emitter;

#[derive(Accounts)]
pub struct InitMessageV1<'info> {
    /// This authority is the only one who can write to the draft message account.
    emitter_authority: Signer<'info>,

    /// CHECK: This account will have been created using the system program outside of the Core
    /// Bridge.
    #[account(
        mut,
        owner = crate::ID
    )]
    draft_message: AccountInfo<'info>,
}

impl<'info> InitMessageV1<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Checking that the message account is completely zeroed out. By doing this, we make the
        // assumption that no other Core Bridge account that is currently used will have all zeros.
        // Ideally all of the Core Bridge accounts should have a discriminator so we do not have to
        // mess around like this. But here we are.
        let msg_acc_data: &[u8] = &ctx.accounts.draft_message.try_borrow_data()?;
        let mut reader = std::io::Cursor::new(msg_acc_data);

        // Infer the expected message length given the size of the created account.
        let data_len = ctx.accounts.draft_message.data_len();
        require_gt!(
            data_len,
            PostedMessageV1::BYTES_START,
            CoreBridgeError::InvalidCreatedAccountSize
        );

        // This message length cannot exceed the maximum message length.
        require_gte!(
            MAX_MESSAGE_PAYLOAD_SIZE,
            data_len - PostedMessageV1::BYTES_START,
            CoreBridgeError::ExceedsMaxPayloadSize
        );

        // All of the discriminator + header bytes + the 4-byte payload length should be zero.
        let mut zeros = [0; PostedMessageV1::BYTES_START];
        reader.read_exact(&mut zeros).unwrap();
        require!(
            zeros == [0; PostedMessageV1::BYTES_START],
            CoreBridgeError::AccountNotZeroed
        );

        // Done.
        Ok(())
    }
}

/// Arguments to initialize a new [PostedMessageV1](crate::state::PostedMessageV1) account for
/// writing.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct InitMessageV1Args {
    pub nonce: u32,
    pub commitment: Commitment,
    pub cpi_program_id: Option<Pubkey>,
}

#[access_control(InitMessageV1::constraints(&ctx))]
pub fn init_message_v1(ctx: Context<InitMessageV1>, args: InitMessageV1Args) -> Result<()> {
    let expected_msg_length = ctx.accounts.draft_message.data_len() - PostedMessageV1::BYTES_START;

    // This message length cannot exceed the maximum message length.
    require_gte!(
        MAX_MESSAGE_PAYLOAD_SIZE,
        expected_msg_length,
        CoreBridgeError::ExceedsMaxPayloadSize
    );

    let InitMessageV1Args {
        nonce,
        commitment,
        cpi_program_id,
    } = args;

    // This instruction allows a program to declare its own program ID as an emitter if we can
    // derive the emitter authority from seeds [b"emitter"]. This is useful for programs that do not
    // want to manage two separate addresses (program ID and emitter address) cross chain.
    let emitter = new_emitter(&ctx.accounts.emitter_authority, cpi_program_id)?;

    let acc_data: &mut [u8] = &mut ctx.accounts.draft_message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);

    // Finally initialize the draft message account by serializing the discriminator, header and
    // payload length.
    writer.write_all(&PostedMessageV1::DISCRIMINATOR)?;
    PostedMessageV1Info {
        consistency_level: commitment.into(),
        emitter_authority: ctx.accounts.emitter_authority.key(),
        status: MessageStatus::Writing,
        _gap_0: Default::default(),
        posted_timestamp: Default::default(),
        nonce,
        sequence: Default::default(),
        solana_chain_id: Default::default(),
        emitter,
    }
    .serialize(&mut writer)?;
    u32::try_from(expected_msg_length)
        .unwrap()
        .serialize(&mut writer)
        .map_err(Into::into)
}
