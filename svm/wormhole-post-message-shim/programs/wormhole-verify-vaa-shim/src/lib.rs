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
        guardian_set_index: u32,
        total_signatures: u8,
        guardian_signatures: Vec<[u8; 66]>,
    ) -> Result<()> {
        instructions::post_signatures(
            ctx,
            guardian_set_index,
            total_signatures,
            guardian_signatures,
        )
    }

    pub fn verify_vaa(ctx: Context<VerifyVaa>, digest: [u8; HASH_BYTES]) -> Result<()> {
        instructions::verify_vaa(ctx, digest)
    }
}
