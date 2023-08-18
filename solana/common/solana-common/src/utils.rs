use anchor_lang::{prelude::*, solana_program};

#[inline]
pub fn is_nonzero_array<const N: usize>(arr: &[u8; N]) -> bool {
    *arr != [0; N]
}

#[inline]
pub fn is_nonzero_slice(slice: &[u8]) -> bool {
    slice.iter().any(|&x| x != 0)
}

pub fn close_account<'info>(
    info: AccountInfo<'info>,
    sol_destination: AccountInfo<'info>,
) -> Result<()> {
    // Transfer tokens from the account to the sol_destination.
    let dest_starting_lamports = sol_destination.lamports();
    **sol_destination.lamports.borrow_mut() =
        dest_starting_lamports.checked_add(info.lamports()).unwrap();
    **info.lamports.borrow_mut() = 0;

    info.assign(&solana_program::system_program::ID);
    info.realloc(0, false).map_err(Into::into)
}

pub fn require_discriminator<const N: usize>(acc_data: &mut &[u8], expected: [u8; N]) -> Result<()>
where
    [u8; N]: AnchorDeserialize,
{
    let discriminator = <[u8; N]>::deserialize(acc_data)?;
    require!(
        discriminator == expected,
        ErrorCode::AccountDiscriminatorMismatch
    );

    Ok(())
}
