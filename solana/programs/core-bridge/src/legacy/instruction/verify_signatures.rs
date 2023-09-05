use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct VerifySignaturesArgs {
    pub signer_indices: [i8; 19],
}
