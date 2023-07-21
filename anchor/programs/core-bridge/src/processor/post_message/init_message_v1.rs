use std::io::{Read, Write};

use crate::{
    constants::MAX_MESSAGE_PAYLOAD_SIZE,
    error::CoreBridgeError,
    state::{MessageStatus, PostedMessageV1, PostedMessageV1Info},
};
use anchor_lang::prelude::*;
use wormhole_solana_common::{utils, LegacyDiscriminator};

use super::new_emitter;

#[derive(Accounts)]
pub struct InitMessageV1<'info> {
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
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        // Checking that the message account is completely zeroed out. By doing this, we make the
        // assumption that no other Core Bridge account that is currently used will have all zeros.
        // Ideally all of the Core Bridge accounts should have a discriminator so we do not have to
        // mess around like this. But here we are.
        let msg_acct_data: &[u8] = &ctx.accounts.draft_message.try_borrow_data()?;
        let mut reader = std::io::Cursor::new(msg_acct_data);

        // All of the discriminator + header bytes + the 4-byte payload length should be zero.
        let mut zeros = [0; PostedMessageV1::BYTES_START];
        reader.read_exact(&mut zeros)?;
        require!(
            !utils::is_nonzero_array(&zeros),
            CoreBridgeError::AccountNotZeroed
        );

        // Done.
        Ok(())
    }
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct InitMessageV1Args {
    pub cpi_program_id: Option<Pubkey>,
}

#[access_control(InitMessageV1::accounts(&ctx))]
pub fn init_message_v1(ctx: Context<InitMessageV1>, args: InitMessageV1Args) -> Result<()> {
    // Infer the expected message length given the size of the created account.
    let expected_msg_length = ctx
        .accounts
        .draft_message
        .data_len()
        .saturating_sub(PostedMessageV1::BYTES_START);
    require_gte!(
        expected_msg_length,
        0,
        CoreBridgeError::InvalidCreatedAccountSize
    );

    // And this message length cannot exceed the maximum message length.
    require_gte!(
        MAX_MESSAGE_PAYLOAD_SIZE,
        expected_msg_length,
        CoreBridgeError::ExceedsMaxPayloadSize
    );

    let InitMessageV1Args { cpi_program_id } = args;

    // This instruction allows a program to declare its own program ID as an emitter if we can
    // derive the emitter authority from seeds [b"emitter"]. This is useful for programs that do not
    // want to manage two separate addresses (program ID and emitter address) cross chain.
    let emitter = new_emitter(&ctx.accounts.emitter_authority, cpi_program_id)?;

    let acct_data: &mut [u8] = &mut ctx.accounts.draft_message.try_borrow_mut_data()?;
    let mut writer = std::io::Cursor::new(acct_data);

    // Finally initialize the draft message account by serializing the discriminator, header and
    // payload length.
    writer.write_all(&PostedMessageV1::LEGACY_DISCRIMINATOR)?;
    PostedMessageV1Info {
        consistency_level: Default::default(),
        emitter_authority: ctx.accounts.emitter_authority.key(),
        status: MessageStatus::Writing,
        _gap_0: Default::default(),
        posted_timestamp: Default::default(),
        nonce: Default::default(),
        sequence: Default::default(),
        solana_chain_id: Default::default(),
        emitter,
    }
    .serialize(&mut writer)?;
    u32::try_from(expected_msg_length)
        .unwrap()
        .serialize(&mut writer)?;

    // Done.
    Ok(())
}
