//! Helpers to derive and read Wormhole accounts.

use {
    borsh::BorshDeserialize,
    bridge::{
        BridgeConfig,
        BridgeData,
        PostedVAAData,
    },
    solana_program::{
        account_info::AccountInfo,
        pubkey::Pubkey,
    },
    wormhole::WormholeError,
};

/// Derive the address of the Wormhole Config account.
pub fn config(id: &Pubkey) -> Pubkey {
    let (config, _) = Pubkey::find_program_address(&[b"Bridge"], id);
    config
}

/// Derive the address of the FeeCollector account.
pub fn fee_collector(id: &Pubkey) -> Pubkey {
    let (fee_collector, _) = Pubkey::find_program_address(&[b"fee_collector"], id);
    fee_collector
}

/// Derive the address of the Sequence account associated with an Emitter.
pub fn sequence(id: &Pubkey, emitter: &Pubkey) -> Pubkey {
    let (sequence, _) = Pubkey::find_program_address(&[b"Sequence", &emitter.to_bytes()], id);
    sequence
}

/// Derive the address of Emitter account capable of signing messages.
pub fn emitter(id: &Pubkey) -> (Pubkey, Vec<&[u8]>, u8) {
    let seeds = &["emitter".as_bytes()];
    let (emitter, bump) = Pubkey::find_program_address(seeds, id);
    (emitter, seeds.to_vec(), bump)
}

/// Deserialize helper the BridgeConfig from a Wormhole config account.
pub fn read_config(config: &AccountInfo) -> Result<BridgeConfig, WormholeError> {
    Ok(BridgeData::try_from_slice(&config.data.borrow())
        .map_err(|_| WormholeError::DeserializeFailed)?
        .config)
}

/// Deserialize helper for parsing from Borsh encoded VAA's from Solana accounts.
pub fn read_vaa(vaa: &AccountInfo) -> Result<PostedVAAData, WormholeError> {
    PostedVAAData::try_from_slice(&vaa.data.borrow()).map_err(|_| WormholeError::DeserializeFailed)
}
