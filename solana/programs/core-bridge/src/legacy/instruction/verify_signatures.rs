use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

/// Argument to verify specific guardian indices.
///
/// NOTE: It is preferred to use the new process of verifying a VAA using the new Core Bridge Anchor
/// instructions. See [init_encoded_vaa](crate::wormhole_core_bridge_solana::init_encoded_vaa) and
/// [process_encoded_vaa](crate::wormhole_core_bridge_solana::process_encoded_vaa) for more info.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct VerifySignaturesArgs {
    pub signer_indices: [i8; 19],
}
