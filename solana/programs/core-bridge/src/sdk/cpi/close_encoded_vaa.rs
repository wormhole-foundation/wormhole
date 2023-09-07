use anchor_lang::prelude::*;

use super::InvokeCoreBridge;

pub trait CloseEncodedVaa<'info>: InvokeCoreBridge<'info> {
    fn write_authority(&self) -> AccountInfo<'info>;

    fn encoded_vaa(&self) -> AccountInfo<'info>;
}

/// SDK method to close an EncodedVaa account. This method will fail if the write authority is not
/// the same one found on the [EncodedVaa](crate::state::EncodedVaa) account.
pub fn close_encoded_vaa<'info, A>(accounts: &A) -> Result<()>
where
    A: CloseEncodedVaa<'info>,
{
    crate::cpi::process_encoded_vaa(
        CpiContext::new(
            accounts.core_bridge_program(),
            crate::cpi::accounts::ProcessEncodedVaa {
                write_authority: accounts.write_authority(),
                encoded_vaa: accounts.encoded_vaa(),
                guardian_set: None,
            },
        ),
        crate::processor::ProcessEncodedVaaDirective::CloseVaaAccount,
    )
}

/// SDK method to close an EncodedVaa account. If the write authority is not the same one found on
/// the [EncodedVaa](crate::state::EncodedVaa) account, this method will return `Ok` with a Solana
/// log message indicating failure.
pub fn maybe_close_encoded_vaa<'info, A>(accounts: &A) -> Result<()>
where
    A: CloseEncodedVaa<'info>,
{
    close_encoded_vaa(accounts).or_else(|_| {
        // Closing the encoded vaa failed, so move on.
        msg!("Cannot close EncodedVaa without Write Authority");

        // Done.
        Ok(())
    })
}
