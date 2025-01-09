use anchor_lang::prelude::*;
use anchor_lang::solana_program::keccak::HASH_BYTES;

declare_id!("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

mod instructions;
pub(crate) use instructions::*;

pub mod state;

pub mod error;

#[program]
pub mod wormhole_verify_vaa_shim {

    use super::*;

    pub fn close_signatures(ctx: Context<CloseSignatures>) -> Result<()> {
        instructions::close_signatures(ctx)
    }

    pub fn post_signatures(
        ctx: Context<PostSignatures>,
        guardian_signatures: Vec<[u8; 66]>,
        total_signatures: u8,
    ) -> Result<()> {
        instructions::post_signatures(ctx, guardian_signatures, total_signatures)
    }

    pub fn verify_vaa(
        ctx: Context<VerifyVaa>,
        digest: [u8; HASH_BYTES],
        guardian_set_index: u32,
    ) -> Result<()> {
        instructions::verify_vaa(ctx, digest, guardian_set_index)
    }
}
