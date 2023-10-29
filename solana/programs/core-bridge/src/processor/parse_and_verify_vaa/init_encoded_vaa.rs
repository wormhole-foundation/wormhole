use crate::{
    error::CoreBridgeError,
    state::{Header, ProcessingStatus},
    zero_copy::EncodedVaa,
};
use anchor_lang::prelude::*;

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
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // The size of the created account must be more than the size of discriminator and header
        // (some VAA buffer > 0 bytes).
        require!(
            ctx.accounts.encoded_vaa.data_len() > EncodedVaa::VAA_START,
            CoreBridgeError::InvalidCreatedAccountSize
        );

        // Check that the encoded VAA account header is completely zeroed out. By doing this, we
        // make the assumption that no other Core Bridge account that is currently used will have
        // all zeros. Ideally all of the Core Bridge accounts should have a discriminator so we do
        // not have to mess around like this. But here we are.
        let msg_acc_data: &[_] = &ctx.accounts.encoded_vaa.try_borrow_data()?;
        require!(
            msg_acc_data[..EncodedVaa::VAA_START] == [0; EncodedVaa::VAA_START],
            CoreBridgeError::AccountNotZeroed
        );

        // Done.
        Ok(())
    }
}

#[access_control(InitEncodedVaa::constraints(&ctx))]
pub fn init_encoded_vaa(ctx: Context<InitEncodedVaa>) -> Result<()> {
    let vaa_len = ctx.accounts.encoded_vaa.data_len() - EncodedVaa::VAA_START;

    let acc_data: &mut [_] = &mut ctx.accounts.encoded_vaa.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);

    // Finally initialize the encoded VAA account by serializing the discriminator, header and
    // expected VAA length.
    (
        EncodedVaa::DISC,
        Header {
            status: ProcessingStatus::Writing,
            write_authority: ctx.accounts.write_authority.key(),
            version: Default::default(),
        },
        u32::try_from(vaa_len).unwrap(),
    )
        .serialize(&mut writer)
        .map_err(Into::into)
}
