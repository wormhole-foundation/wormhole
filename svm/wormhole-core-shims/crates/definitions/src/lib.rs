pub mod solana;

// NOTE: Expand this conditional as Wormhole supports more SVM networks.
cfg_if::cfg_if! {
    if #[cfg(feature = "solana")] {
        pub use solana::*;
    }
}

use solana_program::pubkey::Pubkey;

/// Wormhole Core Bridge program's bridge config account seed.
pub const CORE_BRIDGE_CONFIG_SEED: &[u8] = b"Bridge";

/// Wormhole Core Bridge program's fee collector account seed.
pub const FEE_COLLECTOR_SEED: &[u8] = b"fee_collector";

/// Wormhole Core Bridge program's emitter sequence account seed.
pub const EMITTER_SEQUENCE_SEED: &[u8] = b"Sequence";

/// Anchor event CPI's authority seed.
pub const EVENT_AUTHORITY_SEED: &[u8] = b"__event_authority";

pub fn find_core_bridge_config_address(wormhole_program_id: &Pubkey) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[b"Bridge"], wormhole_program_id)
}

pub fn find_fee_collector_address(wormhole_program_id: &Pubkey) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[b"fee_collector"], wormhole_program_id)
}

pub fn find_emitter_sequence_address(
    emitter: &Pubkey,
    wormhole_program_id: &Pubkey,
) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[b"Sequence", emitter.as_ref()], wormhole_program_id)
}

pub fn find_guardian_set_address(
    index_be_bytes: [u8; 4],
    wormhole_program_id: &Pubkey,
) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[b"GuardianSet", &index_be_bytes], wormhole_program_id)
}

pub fn find_shim_message_address(
    emitter: &Pubkey,
    post_message_program_id: &Pubkey,
) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[emitter.as_ref()], post_message_program_id)
}

pub fn find_event_authority_address(program_id: &Pubkey) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[EVENT_AUTHORITY_SEED], &program_id)
}
