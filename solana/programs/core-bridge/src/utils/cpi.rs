use anchor_lang::{prelude::*, system_program};

/// Trait for invoking the System program's create account instruction.
pub trait CreateAccount<'info> {
    fn system_program(&self) -> AccountInfo<'info>;

    /// Signer that has the lamports to transfer to new account.
    fn payer(&self) -> AccountInfo<'info>;
}

/// Method for invoking the System program's create account instruction. This method may be useful
/// if it is inconvenient to use Anchor's `init` account macro directive.
///
/// NOTE: This method does not serialize any data into your new account. You will need to serialize
/// this data by borrowing mutable data and writing to it.
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
