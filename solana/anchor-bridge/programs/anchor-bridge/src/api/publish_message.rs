use anchor_lang::{prelude::*, solana_program};

use crate::{
    accounts,
    anchor_bridge::Bridge,
    types::{BridgeConfig, Index},
    PublishMessage,
    Result,
    MAX_LEN_GUARDIAN_KEYS,
};

// Constant fee for VAA transactions, measured in lamports.
const VAA_TX_FEE: u64 = 18 * 10000;

pub fn publish_message(bridge: &mut Bridge, ctx: Context<PublishMessage>) -> Result<()> {
    Ok(())
}

// A const time calculation of the fee required to publish a message.
//
// Cost breakdown:
// - 2 Signatures
// - 1 Claimed VAA Rent
// - 2x Guardian Fees
const fn calculate_transfer_fee() -> u64 {
    use std::mem::size_of;
    const SIGNATURE_COST: u64 = size_of::<SignatureState>() as u64;
    const VAA_COST: u64 = size_of::<ClaimedVAA>() as u64;
    const VAA_FEE: u64 = VAA_TX_FEE;
    SIGNATURE_COST + VAA_COST + VAA_FEE
}

/// Signature state
#[repr(C)]
#[derive(Clone, Copy)]
pub struct SignatureState {
    /// signatures of validators
    pub signatures: [[u8; 65]; MAX_LEN_GUARDIAN_KEYS],

    /// hash of the data
    pub hash: [u8; 32],

    /// index of the guardian set
    pub guardian_set_index: u32,
}

/// Record of a claimed VAA
#[repr(C)]
#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct ClaimedVAA {
    /// hash of the vaa
    pub hash: [u8; 32],
    /// time the vaa was submitted
    pub vaa_time: u32,
}
