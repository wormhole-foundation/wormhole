use anchor_lang::{prelude::*, system_program};

/// Trait for invoking the system program's create account instruction.
pub trait CreateAccount<'info> {
    fn system_program(&self) -> AccountInfo<'info>;

    /// Signer that has the lamports to transfer to new account.
    fn payer(&self) -> AccountInfo<'info>;
}

pub fn create_account<'info, A>(
    accounts: &A,
    new_account: &AccountInfo<'info>,
    data_len: usize,
    owner: &Pubkey,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: CreateAccount<'info>,
{
    system_program::create_account(
        CpiContext::new_with_signer(
            accounts.system_program(),
            system_program::CreateAccount {
                from: accounts.payer(),
                to: new_account.to_account_info(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        Rent::get().map(|rent| rent.minimum_balance(data_len))?,
        data_len.try_into().unwrap(),
        owner,
    )
}
