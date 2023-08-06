use anchor_lang::prelude::{
    err, error, require, require_eq, require_keys_eq, ErrorCode, Pubkey, Result,
};

use crate::utils::TruncateAmount;

/// This implements a zero-copy deserialization for the Token Program's mint account. All struct
/// field doc strings are shamelessly copied from the SPL Token docs.
pub struct Mint<'a>(&'a [u8]);

impl<'a> Mint<'a> {
    /// Optional authority used to mint new tokens. The mint authority may only be provided during
    /// mint creation. If no mint authority is present then the mint has a fixed supply and no
    /// further tokens may be minted.
    pub fn mint_authority(&self) -> Option<Pubkey> {
        match u32::from_le_bytes(self.0[..4].try_into().unwrap()) {
            0 => None,
            _ => Some(Pubkey::try_from(&self.0[4..36]).unwrap()),
        }
    }

    pub fn require_mint_authority(
        acc_data: &'a [u8],
        mint_authority: Option<&Pubkey>,
    ) -> Result<()> {
        match (Self::parse(acc_data)?.mint_authority(), mint_authority) {
            (Some(actual), Some(expected)) => {
                require_keys_eq!(actual, *expected, ErrorCode::ConstraintMintMintAuthority);
                Ok(())
            }
            (None, None) => Ok(()),
            _ => err!(ErrorCode::ConstraintMintMintAuthority),
        }
    }

    /// Total supply of tokens.
    pub fn supply(&self) -> u64 {
        u64::from_le_bytes(self.0[36..44].try_into().unwrap())
    }

    /// Number of base 10 digits to the right of the decimal place.
    pub fn decimals(&self) -> u8 {
        self.0[44]
    }

    /// Is `true` if this structure has been initialized
    pub fn is_initialized(&self) -> bool {
        self.0[45] == 1
    }

    /// Optional authority to freeze token accounts.
    pub fn freeze_authority(&self) -> Option<Pubkey> {
        match u32::from_le_bytes(self.0[46..50].try_into().unwrap()) {
            0 => None,
            _ => Some(Pubkey::try_from(&self.0[50..82]).unwrap()),
        }
    }

    pub fn parse(span: &'a [u8]) -> Result<Self> {
        require_eq!(
            span.len(),
            anchor_spl::token::Mint::LEN,
            ErrorCode::AccountDidNotDeserialize
        );

        let mint = Self(span);
        require!(mint.is_initialized(), ErrorCode::AccountNotInitialized);

        Ok(mint)
    }
}

impl<'a> TruncateAmount for Mint<'a> {
    fn mint_decimals(&self) -> u8 {
        self.decimals()
    }
}
