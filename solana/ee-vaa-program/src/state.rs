//! Program account state.
//!
//! This module aims to help manage EE VAA accounts more easily
// use anchor_lang::prelude::*;

use solana_program::{
    account_info::{next_account_info, AccountInfo},
    program_error::ProgramError,
    pubkey::Pubkey,
};

use std::collections::VecDeque;
use std::convert::TryFrom;

use crate::{
    error::{self, Error::*},
    instruction::EEVAA,
};

// #[derive(Accounts)]
/// Common state for EE VAA requests
pub struct EEVAAProgramState {
    queue: VecDeque<EEVAA>,
}

pub struct Call<'a, 'b> {
    pub program_id: &'a Pubkey,
    pub accounts: &'a [AccountInfo<'b>],
}

impl<'a, 'b> TryFrom<Call<'a, 'b>> for EEVAAProgramState {
    type Error = error::Error;
    fn try_from(c: Call) -> Result<Self, Self::Error> {
        let mut acc_iter = c.accounts.iter();

        let queue_acc: &AccountInfo =
            next_account_info(&mut acc_iter).map_err(|_| UnexpectedEndOfBuffer)?;

        // queue_acc
        //     .try_borrow_data()
        //     .map_err(|e| {msg!("Could not borrow queue account data: {:?}", e); })?;

        Ok(EEVAAProgramState {
            queue: VecDeque::new(),
        })
    }
}

pub fn next_account_info_with_owner<'a, 'b, I: Iterator<Item = &'a AccountInfo<'b>>>(
    iter: &mut I,
    owner: &Pubkey,
) -> Result<I::Item, error::Error> {
    let acc = iter.next().ok_or(UnexpectedEndOfAccounts)?;
    if acc.owner != owner {
        return Err(InvalidOwner);
    }
    Ok(acc)
}
