use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyVerifySignaturesArgs {
    pub signer_indices: [i8; 19],
}
