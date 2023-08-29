use anchor_lang::prelude::{
    error, require, require_eq, AnchorDeserialize, ErrorCode, Pubkey, Result,
};

pub struct Mint<'a>(&'a [u8]);

impl<'a> Mint<'a> {
    /// Optional authority used to mint new tokens. The mint authority may only be provided during
    /// mint creation. If no mint authority is present then the mint has a fixed supply and no
    /// further tokens may be minted.
    pub fn mint_authority(&self) -> Option<Pubkey> {
        let mut buf = &self.0[0..36];
        match u32::deserialize(&mut buf).unwrap() {
            0 => None,
            _ => Some(AnchorDeserialize::deserialize(&mut buf).unwrap()),
        }
    }

    /// Total supply of tokens.
    pub fn supply(&self) -> u64 {
        let mut buf = &self.0[36..44];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
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
        let mut buf = &self.0[46..82];
        match u32::deserialize(&mut buf).unwrap() {
            0 => None,
            _ => Some(AnchorDeserialize::deserialize(&mut buf).unwrap()),
        }
    }

    pub fn parse(span: &'a [u8]) -> Result<Self> {
        const LEN: usize = anchor_spl::token::Mint::LEN;
        require_eq!(span.len(), LEN, ErrorCode::AccountDidNotDeserialize);

        let mint = Self(&span[..LEN]);
        require!(mint.is_initialized(), ErrorCode::AccountNotInitialized);

        Ok(mint)
    }
}
