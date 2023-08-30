use crate::{
    constants::{MAX_DECIMALS, MINT_AUTHORITY_SEED_PREFIX},
    error::TokenBridgeError,
    zero_copy::Mint,
};
use anchor_lang::prelude::*;

pub fn require_native_mint(mint: &AccountInfo) -> Result<()> {
    if let Some(mint_authority) = Mint::parse(&mint.try_borrow_data()?)?.mint_authority() {
        let token_bridge_mint_authority =
            Pubkey::find_program_address(&[MINT_AUTHORITY_SEED_PREFIX], &crate::ID).0;
        require_keys_neq!(
            mint_authority,
            token_bridge_mint_authority,
            TokenBridgeError::WrappedAsset
        );
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

impl TruncateAmount for anchor_spl::token::Mint {
    fn mint_decimals(&self) -> u8 {
        self.decimals
    }
}
