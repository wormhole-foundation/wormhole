use crate::{error::TokenBridgeError, state::RegisteredEmitter, ID};
use anchor_lang::prelude::*;
use core_bridge_program::legacy::utils::LegacyAnchorized;
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

/// We disallow certain posted VAA accounts from being used to redeem Token Bridge transfers.
pub fn require_valid_posted_vaa_key(acc_key: &Pubkey) -> Result<()> {
    // IYKYK.
    require!(
        !INVALID_POSTED_VAA_KEYS.contains(&acc_key.to_string().as_str()),
        TokenBridgeError::InvalidTokenBridgeVaa
    );

    Ok(())
}

/// In order for a VAA to be a valid Token Bridge VAA, it must have been sent by another registered
/// Token Bridge and must be serialized as one of the following messages:
/// - Transfer (Payload ID == 1)
/// - Attestation (Payload ID == 2)
/// - Transfer with Message (Payload ID == 3)
pub fn require_valid_posted_token_bridge_vaa<'ctx>(
    vaa_acc_key: &Pubkey,
    vaa: &core_bridge_program::zero_copy::PostedVaaV1<'ctx>,
    registered_emitter: &'ctx Account<'_, LegacyAnchorized<0, RegisteredEmitter>>,
) -> Result<TokenBridgeMessage<'ctx>> {
    require_valid_posted_vaa_key(vaa_acc_key)?;

    let emitter_chain = vaa.emitter_chain();
    let emitter_address = vaa.emitter_address();
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
        let (expected_legacy_address, _) = Pubkey::find_program_address(
            &[&emitter_chain.to_be_bytes(), emitter_address.as_ref()],
            &ID,
        );
        require_keys_eq!(
            emitter_key,
            expected_legacy_address,
            TokenBridgeError::InvalidLegacyTokenBridgeEmitter
        );
    }

    // Make sure we are working with a valid Token Bridge message.
    TokenBridgeMessage::parse(vaa.payload())
        .map_err(|_| error!(TokenBridgeError::CannotParseMessage))
}
