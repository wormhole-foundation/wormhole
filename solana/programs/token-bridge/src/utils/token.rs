use crate::constants::MINT_AUTHORITY_SEED_PREFIX;
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
