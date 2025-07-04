use anchor_lang::{
  prelude::{AccountInfo, CpiContext, Program, Rent, SolanaSysvar, System, ToAccountInfo},
  AccountSerialize,
  Result,
  Space
};

use std::ops::DerefMut;
use crate::ID;


pub trait SeedPrefix {
  const SEED_PREFIX: &'static [u8];
}

pub fn init_account<'info, T: Space + AccountSerialize>(
  account: AccountInfo<'info>,
  seeds: &[&[u8]],
  system_program: &Program<'info, System>,
  rent_payer: AccountInfo<'info>,
  data: T
) -> Result<()> {
  let lamports = account.lamports();
  // Before calling `create_account`, we need to verify that the account
  // has an empty balance, otherwise the instruction would fail:
  if lamports != 0 {
    anchor_lang::system_program::transfer(
      CpiContext::new_with_signer(
        system_program.to_account_info(),
        anchor_lang::system_program::Transfer {
          from: account.clone(),
          to: rent_payer.clone(),
        },
        &[seeds],
      ),
      lamports,
    )?;
  }

  anchor_lang::system_program::create_account(
    CpiContext::new_with_signer(
      system_program.to_account_info(),
      anchor_lang::system_program::CreateAccount {
        from: rent_payer,
        to: account.clone(),
      },
      &[seeds],
    ),
    Rent::get()?.minimum_balance(8 + T::INIT_SPACE),
    (8 + T::INIT_SPACE) as u64,
    &ID,
  )?;

  T::try_serialize(&data, account.try_borrow_mut_data()?.deref_mut())?;
  Ok(())
}