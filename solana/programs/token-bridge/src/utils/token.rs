use crate::{
    constants::{MAX_DECIMALS, MINT_AUTHORITY_SEED_PREFIX},
    error::TokenBridgeError,
    zero_copy::Mint,
};
use anchor_lang::prelude::*;

/// With an account meant to be a Token Program mint account, make sure it is not a mint that the
/// Token Bridge program controls.
pub fn require_native_mint(mint: &AccountInfo) -> Result<()> {
    // This may be redundant because this mint account being owned by the Token Program is
    // associated with either a transfer between two token accounts (which requires that this
    // account be a valid mint) and deriving metadata PDA to create and update token metadata.
    require_eq!(
        *mint.owner,
        anchor_spl::token::ID,
        ErrorCode::ConstraintMintTokenProgram
    );

    // If there is a mint authority, make sure it is not the Token Bridge's mint authority, which
    // controls burn and mint for its wrapped assets.
    if let Some(mint_authority) = Mint::parse(&mint.try_borrow_data()?)?.mint_authority() {
        let (token_bridge_mint_authority, _) =
            Pubkey::find_program_address(&[MINT_AUTHORITY_SEED_PREFIX], &crate::ID);
        require_keys_neq!(
            mint_authority,
            token_bridge_mint_authority,
            TokenBridgeError::WrappedAsset
        );
    }

    // Done.
    Ok(())
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
