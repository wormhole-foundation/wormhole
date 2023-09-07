use crate::{
    constants::{MAX_DECIMALS, MINT_AUTHORITY_SEED_PREFIX},
    zero_copy::Mint,
};
use anchor_lang::prelude::*;

/// Basically check whether the mint authority is the Token Bridge's mint authority.
///
/// NOTE: This method does not guarantee that the mint is a mint created by the Token Bridge program
/// via `create_or_update_wrapped` instruction because someone can transfer mint authority for
/// another mint to the Token Bridge's mint authority.
pub fn is_wrapped_mint(mint: &Mint) -> bool {
    if let Some(mint_authority) = mint.mint_authority() {
        let (token_bridge_mint_authority, _) =
            Pubkey::find_program_address(&[MINT_AUTHORITY_SEED_PREFIX], &crate::ID);
        mint_authority == token_bridge_mint_authority
    } else {
        false
    }
}

/// Convenient trait to determine amount truncation for encoded token transfer amounts.
pub trait TruncateAmount {
    fn mint_decimals(&self) -> u8;

    fn truncate_amount(&self, amount: u64) -> u64 {
        match self.mint_decimals().saturating_sub(MAX_DECIMALS) {
            0 => amount,
            diff => {
                let divisor = u64::pow(10, diff.into());
                (amount / divisor) * divisor
            }
        }
    }
}

impl TruncateAmount for anchor_spl::token::Mint {
    fn mint_decimals(&self) -> u8 {
        self.decimals
    }
}
