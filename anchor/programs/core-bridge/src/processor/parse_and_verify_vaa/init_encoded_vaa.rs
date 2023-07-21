use std::io::Read;

use crate::{
    error::CoreBridgeError,
    state::{EncodedVaa, ProcessingHeader, ProcessingStatus},
};
use anchor_lang::prelude::*;
use wormhole_solana_common::utils;

#[derive(Accounts)]
pub struct InitEncodedVaa<'info> {
    /// The authority who can write to the VAA account when it is being processed.
    write_authority: Signer<'info>,

    /// CHECK: This account will have been created using the system program outside of the Core
    /// Bridge.
    #[account(
        mut,
        owner = crate::ID
    )]
    encoded_vaa: AccountInfo<'info>,
}

impl<'info> InitEncodedVaa<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        // Checking that the message account is completely zeroed out. By doing this, we make the
        // assumption that no other Core Bridge account that is currently used will have all zeros.
        // Ideally all of the Core Bridge accounts should have a discriminator so we do not have to
        // mess around like this. But here we are.
        let msg_acct_data: &[u8] = &ctx.accounts.encoded_vaa.try_borrow_data()?;
        let mut reader = std::io::Cursor::new(msg_acct_data);

        // All of the discriminator + header bytes + the 4-byte payload length should be zero.
        let mut zeros = [0; EncodedVaa::BYTES_START];
        reader.read_exact(&mut zeros)?;
        require!(
            !utils::is_nonzero_array(&zeros),
            CoreBridgeError::AccountNotZeroed
        );

        // Done.
        Ok(())
    }
}

#[access_control(InitEncodedVaa::accounts(&ctx))]
pub fn init_encoded_vaa(ctx: Context<InitEncodedVaa>) -> Result<()> {
    // The size of the created account must be more than the size of discriminator and header
    // (some VAA buffer > 0 bytes).
    let data_len = ctx.accounts.encoded_vaa.data_len();
    require_gt!(
        data_len,
        EncodedVaa::BYTES_START,
        CoreBridgeError::InvalidCreatedAccountSize
    );

    let acct_data: &mut [u8] = &mut ctx.accounts.encoded_vaa.try_borrow_mut_data()?;
    let mut writer = std::io::Cursor::new(acct_data);

    // Finally initialize the encoded VAA account by serializing the discriminator, header and
    // expected VAA length.
    //
    // NOTE: This account layout does not match any account found in the state directory. Only the
    // discriminator and header will match the `VaaV1` account (which is how this account will be
    // serialized once the encoded VAA has finished processing).
    ProcessingHeader {
        status: ProcessingStatus::Writing,
        write_authority: ctx.accounts.write_authority.key(),
        version: Default::default(),
    }
    .try_account_serialize(&mut writer)?;
    u32::try_from(data_len - EncodedVaa::BYTES_START)
        .unwrap()
        .serialize(&mut writer)
        .map_err(Into::into)
}
