mod token;
pub use token::*;

mod vaa;
pub use vaa::*;

use anchor_lang::prelude::*;

/// Determine the sender address based on whether an integrator provides his own program ID. If a
/// program ID is provided, it is verified against the sender authority, which requires specific
/// seeds. Otherwise the sender is the authority itself.
pub fn new_sender_address(
    sender_authority: &Signer,
    cpi_program_id: Option<Pubkey>,
) -> Result<Pubkey> {
    match cpi_program_id {
        Some(program_id) => {
            let (expected_authority, _) = Pubkey::find_program_address(
                &[crate::constants::PROGRAM_SENDER_SEED_PREFIX],
                &program_id,
            );
            require_eq!(sender_authority.key(), expected_authority);
            Ok(program_id)
        }
        None => Ok(sender_authority.key()),
    }
}
