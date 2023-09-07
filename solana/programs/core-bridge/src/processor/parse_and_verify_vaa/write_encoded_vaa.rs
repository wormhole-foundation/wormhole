use crate::{error::CoreBridgeError, state::ProcessingStatus, zero_copy::EncodedVaa};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct WriteEncodedVaa<'info> {
    /// The only authority that can write to the encoded VAA account.
    write_authority: Signer<'info>,

    /// CHECK: The encoded VAA account, which stores the VAA buffer. This buffer must first be
    /// written to and then verified.
    #[account(mut)]
    encoded_vaa: AccountInfo<'info>,
}

impl<'info> WriteEncodedVaa<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Check write authority.
        let vaa = EncodedVaa::parse_unverified(&ctx.accounts.encoded_vaa)?;
        require_keys_eq!(
            ctx.accounts.write_authority.key(),
            vaa.write_authority(),
            CoreBridgeError::WriteAuthorityMismatch
        );

        // Done.
        Ok(())
    }
}

/// Arguments for the [write_encoded_vaa](crate::wormhole_core_bridge_solana::write_encoded_vaa)
/// instruction.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct WriteEncodedVaaArgs {
    /// Index of VAA buffer.
    pub index: u32,
    /// Data representing subset of VAA buffer starting at specified index.
    pub data: Vec<u8>,
}

#[access_control(WriteEncodedVaa::constraints(&ctx))]
pub fn write_encoded_vaa(ctx: Context<WriteEncodedVaa>, args: WriteEncodedVaaArgs) -> Result<()> {
    let WriteEncodedVaaArgs { index, data } = args;

    require!(
        !data.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    let vaa_size: usize = {
        let vaa = EncodedVaa::parse_unverified(&ctx.accounts.encoded_vaa).unwrap();
        require!(
            vaa.status() == ProcessingStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        vaa.vaa_size()
    };

    let index = usize::try_from(index).unwrap();
    let end = index.saturating_add(data.len());
    require!(end <= vaa_size, CoreBridgeError::DataOverflow);

    let acc_data: &mut [_] = &mut ctx.accounts.encoded_vaa.data.borrow_mut();
    acc_data[(EncodedVaa::VAA_START + index)..(EncodedVaa::VAA_START + end)].copy_from_slice(&data);

    // Done.
    Ok(())
}
