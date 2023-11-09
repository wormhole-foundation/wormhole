use crate::{error::CoreBridgeError, state::EncodedVaa};
use anchor_lang::prelude::*;
use solana_program::program_memory::sol_memcpy;

#[derive(Accounts)]
pub struct WriteEncodedVaa<'info> {
    /// The only authority that can write to the encoded VAA account.
    write_authority: Signer<'info>,

    /// CHECK: The encoded VAA account, which stores the VAA buffer. This buffer must first be
    /// written to and then verified.
    #[account(
        mut,
        owner = crate::ID,
        constraint = EncodedVaa::require_draft_vaa(&draft_vaa, &write_authority)?
    )]
    draft_vaa: AccountInfo<'info>,
}

impl<'info> WriteEncodedVaa<'info> {
    fn constraints(ctx: &Context<Self>, args: &WriteEncodedVaaArgs) -> Result<()> {
        require!(
            !args.data.is_empty(),
            CoreBridgeError::InvalidInstructionArgument
        );

        let msg_length = EncodedVaa::payload_size_unsafe(&ctx.accounts.draft_vaa.data.borrow());

        require!(
            args.index
                .saturating_add(args.data.len().try_into().unwrap())
                <= msg_length,
            CoreBridgeError::DataOverflow
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

#[access_control(WriteEncodedVaa::constraints(&ctx, &args))]
pub fn write_encoded_vaa(ctx: Context<WriteEncodedVaa>, args: WriteEncodedVaaArgs) -> Result<()> {
    let WriteEncodedVaaArgs { index, data } = args;

    let acc_data: &mut [_] = &mut ctx.accounts.draft_vaa.data.borrow_mut();
    sol_memcpy(
        &mut acc_data[(EncodedVaa::VAA_START + usize::try_from(index).unwrap())..],
        &data,
        data.len(),
    );

    // Done.
    Ok(())
}
