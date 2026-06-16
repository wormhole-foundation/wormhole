//! Shared PDA initialisation helper.
//!
//! Defends against the dust-DoS grief vector (an attacker pre-funds the PDA
//! address so a naive `CreateAccount` would fail). Three branches:
//!
//! 1. Empty + zero-lamport + system-owned -> `CreateAccount`.
//! 2. Pre-funded + system-owned + data-empty -> Transfer (top up if short) +
//!    Allocate + Assign, with an explicit owner check surfacing `InvalidPda`.
//! 3. Anything else -> `InvalidPda`.

use pinocchio::{
    cpi::Signer,
    sysvars::{rent::Rent, Sysvar},
    AccountView, Address, ProgramResult,
};
use pinocchio_system::instructions::{Allocate, Assign, CreateAccount, Transfer};

use crate::definitions::GlobalAccountantError;
use crate::err;

pub fn init_or_upgrade_pda(
    payer: &AccountView,
    pda: &AccountView,
    program_id: &Address,
    signer: Signer,
    space: u64,
) -> ProgramResult {
    let rent_exempt_minimum = Rent::get()?.try_minimum_balance(space as usize)?;
    let initial_lamports = pda.lamports();
    let initial_data_len = pda.data_len();
    let initial_owner_is_system = pda.owner() == &pinocchio_system::ID;

    if initial_data_len != 0 || !initial_owner_is_system {
        return Err(err(GlobalAccountantError::InvalidPda));
    }

    if initial_lamports == 0 {
        CreateAccount {
            from: payer,
            to: pda,
            lamports: rent_exempt_minimum,
            space,
            owner: program_id,
        }
        .invoke_signed(core::slice::from_ref(&signer))?;
    } else {
        // Over-funded PDA is accepted; `saturating_sub` keeps `top_up` at 0.
        let top_up = rent_exempt_minimum.saturating_sub(initial_lamports);
        if top_up > 0 {
            Transfer {
                from: payer,
                to: pda,
                lamports: top_up,
            }
            .invoke()?;
        }
        Allocate {
            account: pda,
            space,
        }
        .invoke_signed(core::slice::from_ref(&signer))?;
        Assign {
            account: pda,
            owner: program_id,
        }
        .invoke_signed(core::slice::from_ref(&signer))?;
    }

    Ok(())
}
