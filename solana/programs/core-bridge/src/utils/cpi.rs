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
    // If the account being initialized already has lamports, then we need to send an amount of
    // lamports to the account to cover rent, allocate space and then assign to the owner.
    // Otherwise, we use the create account instruction.
    //
    // NOTE: This was taken from Anchor's create account handling.
    let current_lamports = new_account.lamports();
    if current_lamports == 0 {
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
    } else {
        allocate_and_assign_account(
            accounts,
            new_account,
            data_len,
            owner,
            current_lamports,
            signer_seeds,
        )
    }
}

fn allocate_and_assign_account<'info, A>(
    accounts: &A,
    new_account: &AccountInfo<'info>,
    data_len: usize,
    owner: &Pubkey,
    current_lamports: u64,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: CreateAccount<'info>,
{
    // Fund the account for rent exemption.
    let required_lamports = Rent::get().map(|rent| {
        rent.minimum_balance(data_len)
            .saturating_sub(current_lamports)
    })?;
    if required_lamports > 0 {
        system_program::transfer(
            CpiContext::new(
                accounts.system_program(),
                system_program::Transfer {
                    from: accounts.payer(),
                    to: new_account.to_account_info(),
                },
            ),
            required_lamports,
        )?;
    }

    // Allocate space.
    system_program::allocate(
        CpiContext::new_with_signer(
            accounts.system_program(),
            system_program::Allocate {
                account_to_allocate: new_account.to_account_info(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        data_len.try_into().unwrap(),
    )?;

    // Assign to the owner.
    system_program::assign(
        CpiContext::new_with_signer(
            accounts.system_program(),
            system_program::Assign {
                account_to_assign: new_account.to_account_info(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        owner,
    )
}
