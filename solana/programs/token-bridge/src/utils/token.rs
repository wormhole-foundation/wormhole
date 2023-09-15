use crate::constants::{MAX_DECIMALS, MINT_AUTHORITY_SEED_PREFIX};
use anchor_lang::prelude::*;
use anchor_spl::token::Mint;
use solana_program::program_option::COption;

pub fn require_native_mint(mint: &Mint) -> Result<()> {
    if let COption::Some(mint_authority) = mint.mint_authority {
        let (token_bridge_mint_authority, _) =
            Pubkey::find_program_address(&[MINT_AUTHORITY_SEED_PREFIX], &crate::ID);
        require_keys_neq!(mint_authority, token_bridge_mint_authority);
    }

    Ok(())
}

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

impl TruncateAmount for Mint {
    fn mint_decimals(&self) -> u8 {
        self.decimals
    }
}
