use crate::zero_copy::VaaAccount;
use anchor_lang::{prelude::*, system_program};

/// Trait for invoking the system program's create account instruction.
pub trait CreateAccount<'info> {
    fn system_program(&self) -> AccountInfo<'info>;

    /// Signer that has the lamports to transfer to new account.
    fn payer(&self) -> AccountInfo<'info>;
}

pub fn create_account<'info, A>(
    accounts: &A,
    new_account: AccountInfo<'info>,
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
                to: new_account,
            },
            signer_seeds.unwrap_or_default(),
        ),
        Rent::get().map(|rent| rent.minimum_balance(data_len))?,
        data_len.try_into().unwrap(),
        owner,
    )
}

/// This method provides a way to prevent replay attacks on VAAs. It creates a PDA for your program
/// using seeds \[emitter_address, emitter_chain, sequence\]. By calling this method, it creates an
/// account of one byte (storing the bump of this PDA address). If your instruction handler is
/// called again, this step will fail because the account will already exist.
pub fn claim_vaa<'a, 'info, A>(
    accounts: &A,
    program_id: &Pubkey,
    vaa: &'a VaaAccount<'a>,
    claim_acc_info: &'a AccountInfo<'info>,
) -> Result<()>
where
    A: CreateAccount<'info>,
{
    let (emitter_address, emitter_chain, sequence) = {
        let (addr, chain, seq) = vaa.try_emitter_info()?;
        (addr, chain.to_be_bytes(), seq.to_be_bytes())
    };

    // First make sure the claim address is derived as what we expect.
    let (expected_addr, claim_bump) = Pubkey::find_program_address(
        &[
            emitter_address.as_ref(),
            emitter_chain.as_ref(),
            sequence.as_ref(),
        ],
        program_id,
    );
    require_keys_eq!(
        claim_acc_info.key(),
        expected_addr,
        ErrorCode::ConstraintSeeds
    );

    // In the legacy implementation, claim accounts stored a boolean (1 byte). Instead, we repurpose
    // this account to store something a little more useful: the bump of the PDA address.
    create_account(
        accounts,
        claim_acc_info.to_account_info(),
        1,
        program_id,
        Some(&[&[
            emitter_address.as_ref(),
            emitter_chain.as_ref(),
            sequence.as_ref(),
            &[claim_bump],
        ]]),
    )?;

    claim_acc_info.data.borrow_mut()[0] = claim_bump;

    // Done.
    Ok(())
}
