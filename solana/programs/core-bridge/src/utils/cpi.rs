use anchor_lang::{prelude::*, system_program};

#[derive(Accounts)]
pub struct CreateAccountSafe<'info> {
    /// Payer (mut signer).
    ///
    /// CHECK: This account's lamports will be used to create the new account.
    pub payer: AccountInfo<'info>,

    /// New account (mut).
    ///
    /// CHECK: This account will be created by the System program.
    pub new_account: AccountInfo<'info>,
}

/// Method for invoking the System program's create account instruction. This method may be useful
/// if it is inconvenient to use Anchor's `init` account macro directive.
///
/// NOTE: This method does not serialize any data into your new account. You will need to serialize
/// this data by borrowing mutable data and writing to it.
pub fn create_account_safe<'info>(
    ctx: CpiContext<'_, '_, '_, 'info, CreateAccountSafe<'info>>,
    data_len: usize,
    owner: &Pubkey,
) -> Result<()> {
    // If the account being initialized already has lamports, then we need to send an amount of
    // lamports to the account to cover rent, allocate space and then assign to the owner.
    // Otherwise, we use the create account instruction.
    //
    // NOTE: This was taken from Anchor's create account handling.
    let current_lamports = ctx.accounts.new_account.lamports();
    if current_lamports == 0 {
        system_program::create_account(
            CpiContext::new_with_signer(
                ctx.program,
                system_program::CreateAccount {
                    from: ctx.accounts.payer,
                    to: ctx.accounts.new_account,
                },
                ctx.signer_seeds,
            ),
            Rent::get().map(|rent| rent.minimum_balance(data_len))?,
            data_len.try_into().unwrap(),
            owner,
        )
    } else {
        allocate_and_assign_account(ctx, data_len, owner, current_lamports)
    }
}

fn allocate_and_assign_account<'info>(
    ctx: CpiContext<'_, '_, '_, 'info, CreateAccountSafe<'info>>,
    data_len: usize,
    owner: &Pubkey,
    current_lamports: u64,
) -> Result<()> {
    // Fund the account for rent exemption.
    let required_lamports = Rent::get().map(|rent| {
        rent.minimum_balance(data_len)
            .saturating_sub(current_lamports)
    })?;
    if required_lamports > 0 {
        system_program::transfer(
            CpiContext::new(
                ctx.program.to_account_info(),
                system_program::Transfer {
                    from: ctx.accounts.payer,
                    to: ctx.accounts.new_account.to_account_info(),
                },
            ),
            required_lamports,
        )?;
    }

    // Allocate space.
    system_program::allocate(
        CpiContext::new_with_signer(
            ctx.program.to_account_info(),
            system_program::Allocate {
                account_to_allocate: ctx.accounts.new_account.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        data_len.try_into().unwrap(),
    )?;

    // Assign to the owner.
    system_program::assign(
        CpiContext::new_with_signer(
            ctx.program,
            system_program::Assign {
                account_to_assign: ctx.accounts.new_account.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        owner,
    )
}
