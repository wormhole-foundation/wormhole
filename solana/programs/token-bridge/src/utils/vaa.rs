use crate::{error::TokenBridgeError, state::RegisteredEmitter, ID};
use anchor_lang::prelude::*;
use core_bridge_program::state::PostedVaaV1Bytes;
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

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

pub(crate) fn require_valid_token_bridge_posted_vaa<'ctx>(
    vaa: &'ctx Account<'_, PostedVaaV1Bytes>,
    registered_emitter: &'ctx Account<'info, RegisteredEmitter>,
) -> Result<TokenBridgeMessage<'ctx>> {
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

    let span: &[u8] = vaa.payload.as_ref();

    // Done.
    TokenBridgeMessage::parse(span).map_err(|_| TokenBridgeError::CannotParseMessage.into())
}
