use anchor_lang::{prelude::*, solana_program};

use super::{AccountBump, RequireAuthority};

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

/// This method exists in case an account that requires an emitter is created using
/// `init_if_needed`. This behavior may be a bug, but `has_one` in the `account` macro checks
/// whether the struct member is set to the correct key even when the account is just created.
pub fn check_init_if_needed_authority<T>(acct: &mut Account<T>, authority: &Pubkey) -> Result<()>
where
    T: AccountSerialize + AccountDeserialize + Clone + RequireAuthority,
{
    if acct.authority_key() == Default::default() {
        acct.set_authority(authority);
    } else {
        require_keys_eq!(*authority, acct.authority_key());
    }

    Ok(())
}

/// Set bump if unset. This method determines whether the bump seed is unset only if it equals
/// zero, which is an unlikely event. But if it is, your PDA address derivation requires
/// slightly more compute units than everyone else's because of the `find_program_address` call.
pub fn check_init_if_needed_pda_address<T>(acct: &mut Account<T>) -> Result<()>
where
    T: AccountSerialize + AccountDeserialize + Clone + Owner + AccountBump,
{
    let pda_address = match acct.bump_seed() {
        0 => {
            let (addr, bump) = T::find_program_address(acct);
            acct.set_bump_seed(bump);

            addr
        }
        _ => T::create_program_address(acct).map_err(|_| ErrorCode::ConstraintSeeds)?,
    };

    require_keys_eq!(pda_address, acct.key());

    Ok(())
}

pub fn require_discriminator<const N: usize>(acct_data: &mut &[u8], expected: [u8; N]) -> Result<()>
where
    [u8; N]: AnchorDeserialize,
{
    let discriminator = <[u8; N]>::deserialize(acct_data)?;
    require!(
        discriminator == expected,
        ErrorCode::AccountDiscriminatorMismatch
    );

    Ok(())
}
