use crate::{
    constants::{MAX_DECIMALS, MINT_AUTHORITY_SEED_PREFIX},
    error::TokenBridgeError,
    zero_copy::Mint,
};
use anchor_lang::prelude::*;
use core_bridge_program::sdk::LoadZeroCopy;

/// With an account meant to be a Token Program mint account, make sure it is not a mint that the
/// Token Bridge program controls.
pub fn require_native_mint(acc_info: &AccountInfo) -> Result<()> {
    // If there is a mint authority, make sure it is not the Token Bridge's mint authority, which
    // controls burn and mint for its wrapped assets.
    let mint = Mint::load(acc_info)?;
    require!(is_native_mint(&mint), TokenBridgeError::WrappedAsset);

    // Done.
    Ok(())
}

pub fn is_native_mint(mint: &Mint) -> bool {
    if let Some(mint_authority) = mint.mint_authority() {
        let (token_bridge_mint_authority, _) =
            Pubkey::find_program_address(&[MINT_AUTHORITY_SEED_PREFIX], &crate::ID);
        mint_authority != token_bridge_mint_authority
    } else {
        true
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
