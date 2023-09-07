use anchor_lang::prelude::*;

pub trait CloseEncodedVaa<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info>;

    fn write_authority(&self) -> AccountInfo<'info>;

    fn encoded_vaa(&self) -> AccountInfo<'info>;
}

/// SDK method to close an EncodedVaa account. This method will fail if the write authority is not
/// the same one found on the [EncodedVaa](crate::state::EncodedVaa) account.
pub fn close_encoded_vaa<'info, A>(accounts: &A) -> Result<()>
where
    A: CloseEncodedVaa<'info>,
{
    crate::cpi::close_encoded_vaa(CpiContext::new(
        accounts.core_bridge_program(),
        crate::cpi::accounts::CloseEncodedVaa {
            write_authority: accounts.write_authority(),
            encoded_vaa: accounts.encoded_vaa(),
        },
    ))
}
