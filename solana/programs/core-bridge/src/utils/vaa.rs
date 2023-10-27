use crate::zero_copy::VaaAccount;
use anchor_lang::prelude::*;

/// This method provides a way to prevent replay attacks on VAAs. It creates a PDA for your program
/// using seeds \[emitter_address, emitter_chain, sequence\]. By calling this method, it creates an
/// account of one byte (storing the bump of this PDA address). If your instruction handler is
/// called again, this step will fail because the account will already exist.
pub fn claim_vaa<'info, A>(
    accounts: &A,
    claim_acc_info: &AccountInfo<'info>,
    program_id: &Pubkey,
    vaa: &VaaAccount,
    claim_seed_prefix: Option<&[u8]>,
) -> Result<()>
where
    A: super::cpi::CreateAccount<'info>,
{
    let (emitter_address, emitter_chain, sequence) = {
        let (addr, chain, seq) = vaa.try_emitter_info()?;
        (addr, chain.to_be_bytes(), seq.to_be_bytes())
    };

    // First make sure the claim address is derived as what we expect.
    let (expected_addr, claim_bump) = match claim_seed_prefix {
        Some(prefix) => Pubkey::find_program_address(
            &[
                prefix,
                emitter_address.as_ref(),
                emitter_chain.as_ref(),
                sequence.as_ref(),
            ],
            program_id,
        ),
        None => Pubkey::find_program_address(
            &[
                emitter_address.as_ref(),
                emitter_chain.as_ref(),
                sequence.as_ref(),
            ],
            program_id,
        ),
    };
    require_keys_eq!(
        claim_acc_info.key(),
        expected_addr,
        ErrorCode::ConstraintSeeds
    );

    // In the legacy implementation, claim accounts stored a boolean (1 byte). Instead, we repurpose
    // this account to store something a little more useful: the bump of the PDA address.
    super::cpi::create_account(
        accounts,
        claim_acc_info,
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
