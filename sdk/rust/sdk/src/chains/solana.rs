use borsh::BorshDeserialize;
use solana_program::account_info::AccountInfo;
use solana_program::entrypoint::ProgramResult;
use solana_program::program::invoke_signed;
use solana_program::pubkey::Pubkey;
use std::str::FromStr;

// Export Bridge API
pub use bridge::types::ConsistencyLevel;
pub use bridge::{
    instructions,
    solitaire as bridge_entrypoint,
    BridgeConfig,
    BridgeData,
    MessageData,
    PostVAAData,
    PostedVAAData,
    VerifySignaturesData,
};

use wormhole_core::{
    WormholeError,
    VAA,
};

/// Export Core Mainnet Contract Address
#[cfg(feature = "mainnet")]
pub fn id() -> Pubkey {
    Pubkey::from_str("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth").unwrap()
}

/// Export Core Devnet Contract Address
#[cfg(feature = "testnet")]
pub fn id() -> Pubkey {
    Pubkey::from_str("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5").unwrap()
}

/// Export Local Tilt Devnet Contract Address
#[cfg(feature = "devnet")]
pub fn id() -> Pubkey {
    Pubkey::from_str("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o").unwrap()
}

/// Derives the Wormhole configuration account address.
pub fn config(id: &Pubkey) -> Pubkey {
    let (config, _) = Pubkey::find_program_address(&[b"Bridge"], &id);
    config
}

/// Derives the Wormhole fee account address, users of the bridge must pay this address before
/// submitting messages to the bridge.
pub fn fee_collector(id: &Pubkey) -> Pubkey {
    let (fee_collector, _) = Pubkey::find_program_address(&[b"fee_collector"], &id);
    fee_collector
}

/// Derives the sequence address for an emitter, which is incremented after each message post.
pub fn sequence(id: &Pubkey, emitter: &Pubkey) -> Pubkey {
    let (sequence, _) = Pubkey::find_program_address(&[b"Sequence", &emitter.to_bytes()], &id);
    sequence
}

/// Derives the emitter address for a Solana contract, the emitter on Solana must be a signer, this
/// function helps generate a PDA and bump seed so users can emit using a PDA as the emitter.
pub fn emitter(id: &Pubkey) -> (Pubkey, Vec<&[u8]>, u8) {
    let seeds = &["emitter".as_bytes()];
    let (emitter, bump) = Pubkey::find_program_address(seeds, id);
    (emitter, seeds.to_vec(), bump)
}

/// Deserialize helper the BridgeConfig from a Wormhole config account.
pub fn read_config(config: &AccountInfo) -> Result<BridgeConfig, WormholeError> {
    let bridge_data = BridgeData::try_from_slice(&config.data.borrow())
        .map_err(|_| WormholeError::DeserializeFailed)?;
    Ok(bridge_data.config)
}

/// Deserialize helper for parsing from Borsh encoded VAA's from Solana accounts.
pub fn read_vaa(vaa: &AccountInfo) -> Result<PostedVAAData, WormholeError> {
    Ok(PostedVAAData::try_from_slice(&vaa.data.borrow())
        .map_err(|_| WormholeError::DeserializeFailed)?)
}

/// This helper method wraps the steps required to invoke Wormhole, it takes care of fee payment,
/// emitter derivation, and function invocation. This will be the right thing to use if you need to
/// simply emit a message in the most straight forward way possible.
pub fn post_message(
    program_id: Pubkey,
    payer: Pubkey,
    message: Pubkey,
    payload: impl AsRef<[u8]>,
    consistency: ConsistencyLevel,
    seeds: Option<&[&[u8]]>,
    accounts: &[AccountInfo],
    nonce: u32,
) -> ProgramResult {
    post_message_internal(
        program_id,
        payer,
        message,
        payload,
        consistency,
        seeds,
        accounts,
        nonce,
        true,
    )
}

/// This helper method wraps the steps required to invoke Wormhole, it takes care of fee payment,
/// emitter derivation, and function invocation.
/// This posts a message without persisting it on Solana. This means this message might be dropped
/// and can not be recovered/reprocessed.
/// DO NOT USE THIS FOR CRITICAL MESSAGES LIKE TRANSFER OF FUNDS
pub fn post_message_unreliable(
    program_id: Pubkey,
    payer: Pubkey,
    message: Pubkey,
    payload: impl AsRef<[u8]>,
    consistency: ConsistencyLevel,
    seeds: Option<&[&[u8]]>,
    accounts: &[AccountInfo],
    nonce: u32,
) -> ProgramResult {
    post_message_internal(
        program_id,
        payer,
        message,
        payload,
        consistency,
        seeds,
        accounts,
        nonce,
        false,
    )
}

fn post_message_internal(
    program_id: Pubkey,
    payer: Pubkey,
    message: Pubkey,
    payload: impl AsRef<[u8]>,
    consistency: ConsistencyLevel,
    seeds: Option<&[&[u8]]>,
    accounts: &[AccountInfo],
    nonce: u32,
    persist: bool,
) -> ProgramResult {
    // Derive any necessary Pubkeys, derivation makes sure that we match the accounts the are being
    // provided by the user as well.
    let id = id();
    let fee_collector = fee_collector(&id);
    let (emitter, mut emitter_seeds, bump) = emitter(&program_id);
    let bump = &[bump];
    emitter_seeds.push(bump);

    // Filter for the Config AccountInfo so we can access its data.
    let config = config(&id);
    let config = accounts.iter().find(|item| *item.key == config).unwrap();
    let config = read_config(config).unwrap();

    // Pay Fee to the Wormhole
    invoke_signed(
        &solana_program::system_instruction::transfer(&payer, &fee_collector, config.fee),
        accounts,
        &[],
    )?;

    // Invoke the Wormhole post_message endpoint to create an on-chain message.
    invoke_signed(
        &if persist {
            instructions::post_message(
                id,
                payer,
                emitter,
                message,
                nonce,
                payload.as_ref().to_vec(),
                consistency,
            )
        } else {
            instructions::post_message_unreliable(
                id,
                payer,
                emitter,
                message,
                nonce,
                payload.as_ref().to_vec(),
                consistency,
            )
        }
        .unwrap(),
        accounts,
        &[&emitter_seeds, seeds.unwrap_or(&[])],
    )?;

    Ok(())
}
