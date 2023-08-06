mod asset_metadata;
mod governance;
mod token_transfer;
mod token_transfer_with_payload;

pub use asset_metadata::*;
pub(crate) use governance::*;
pub use token_transfer::*;
pub use token_transfer_with_payload::*;

use anchor_lang::prelude::*;
use core_bridge_program::{state::PostedVaaV1, WormDecode, WormEncode};

use crate::{error::TokenBridgeError, state::RegisteredEmitter, ID};

// Static list of invalid VAA Message accounts.
const INVALID_POSTED_VAA_KEYS: [&str; 7] = [
    "28Tx7c3W8rggVNyUQEAL9Uq6pUng4xJLAeLA6V8nLH1Z",
    "32YEuzLCvSyHoV6NFpaTXfiAB8sHiAnYcvP2BBeLeGWq",
    "427N2RrDHYooLvyWCiEiNR4KtGsGFTMuXiGwtuChWRSd",
    "56Vf4Y2SCxJBf4TSR24fPF8qLHhC8ZuTJvHS6mLGWieD",
    "7SzK4pmh9fM9SWLTCKmbjQC8EvDgPmtwdaBeTRztkM98",
    "G2VJNjmQsz6wfVZkTUzYAB8ZzRS2hZbpUd5Cr4DTpz6t",
    "GvAarWUV8khMLrTRouzBh3xSr8AeLDXxoKNJ6FgxGyg5",
];

pub trait TokenBridgeMessage: wormhole_vaas::Payload {}

pub(crate) fn require_valid_token_bridge_posted_vaa<'ctx, T: TokenBridgeMessage>(
    vaa: &'ctx Account<'_, PostedVaaV1<T>>,
    registered_emitter: &'ctx Account<'_, RegisteredEmitter>,
) -> Result<&'ctx T> {
    // IYKYK.
    require!(
        !INVALID_POSTED_VAA_KEYS.contains(&vaa.key().to_string().as_str()),
        TokenBridgeError::InvalidPostedVaa
    );

    let emitter_chain = vaa.emitter_chain;
    let emitter_address = vaa.emitter_address;
    let emitter_key = registered_emitter.key();

    // Validate registered emitter PDA address.
    //
    // NOTE: We can move the PDA address check back into the Anchor account context macro once we
    // have migrated all registered emitter accounts to the new seeds.
    let (derived_emitter, _) = Pubkey::find_program_address(&[&emitter_chain.to_be_bytes()], &ID);
    if emitter_key == derived_emitter {
        // Emitter info must agree with our registered foreign Token Bridge.
        require!(
            emitter_address == registered_emitter.contract,
            TokenBridgeError::InvalidTokenBridgeEmitter
        )
    } else {
        // If the legacy definition, the seeds define the contents of this account.
        let (legacy_address, _) = Pubkey::find_program_address(
            &[&emitter_chain.to_be_bytes(), emitter_address.as_ref()],
            &ID,
        );
        require_keys_eq!(emitter_key, legacy_address);
    }

    // Done.
    Ok(&vaa.payload)
}
