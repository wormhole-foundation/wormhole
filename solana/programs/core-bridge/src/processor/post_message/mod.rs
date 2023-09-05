mod init_message_v1;
pub use init_message_v1::*;

mod process_message_v1;
pub use process_message_v1::*;

use crate::error::CoreBridgeError;
use anchor_lang::prelude::*;

pub fn new_emitter(emitter_authority: &Signer, cpi_program_id: Option<Pubkey>) -> Result<Pubkey> {
    match cpi_program_id {
        Some(program_id) => {
            let (expected_authority, _) = Pubkey::find_program_address(
                &[crate::constants::PROGRAM_EMITTER_SEED_PREFIX],
                &program_id,
            );
            require_eq!(
                emitter_authority.key(),
                expected_authority,
                CoreBridgeError::InvalidProgramEmitter
            );
            Ok(program_id)
        }
        None => Ok(emitter_authority.key()),
    }
}
