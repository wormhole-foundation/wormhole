pub mod solana;

// NOTE: Expand this conditional as Wormhole supports more SVM networks.
cfg_if::cfg_if! {
    if #[cfg(feature = "solana")] {
        pub use solana::*;
    }
}

pub const MESSAGE_AUTHORITY_SEED: &[u8] = b"__event_authority";

// NOTE: Expand this as #[cfg(or(...))] as Wormhole supports more SVM networks.
#[cfg(feature = "solana")]
mod pda {
    use solana_program::pubkey::Pubkey;

    use super::*;

    pub fn find_emitter_sequence_address(emitter: &Pubkey) -> (Pubkey, u8) {
        Pubkey::find_program_address(&[b"Sequence", emitter.as_ref()], &CORE_BRIDGE_PROGRAM_ID)
    }

    pub fn find_message_authority_address(program_id: &Pubkey) -> (Pubkey, u8) {
        Pubkey::find_program_address(&[MESSAGE_AUTHORITY_SEED], &program_id)
    }

    pub fn find_shim_message_address(emitter: &Pubkey) -> (Pubkey, u8) {
        Pubkey::find_program_address(&[emitter.as_ref()], &POST_MESSAGE_SHIM_PROGRAM_ID)
    }
}

#[cfg(feature = "solana")]
pub use pda::*;
