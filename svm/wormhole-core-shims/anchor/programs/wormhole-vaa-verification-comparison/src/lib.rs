use anchor_lang::prelude::*;

declare_id!("m7qfXCwkwvpB4RayZoVsNxm4X2iE2a34rCAsjbNvrRs");

mod instructions;
pub(crate) use instructions::*;

pub mod state;

pub mod error;

#[program]
pub mod wormhole_vaa_verification_comparison {
    use super::*;

    pub fn consume_core_posted_vaa(
        ctx: Context<ConsumeCorePostedVaa>,
        vaa_hash: [u8; 32],
    ) -> Result<()> {
        instructions::consume_core_posted_vaa(ctx, vaa_hash)
    }

    pub fn post_signatures(
        ctx: Context<PostSignatures>,
        guardian_signatures: Vec<[u8; 66]>,
        total_signatures: u8,
    ) -> Result<()> {
        instructions::post_signatures(ctx, guardian_signatures, total_signatures)
    }

    pub fn consume_vaa(
        ctx: Context<ConsumeVaa>,
        vaa_body: Vec<u8>,
        guardian_set_index: u32,
    ) -> Result<()> {
        instructions::consume_vaa(ctx, vaa_body, guardian_set_index)
    }

    pub fn consume_vaa_via_shim(
        ctx: Context<ConsumeVaaViaShim>,
        guardian_set_bump: u8,
        vaa_body: Vec<u8>,
    ) -> Result<()> {
        instructions::consume_vaa_via_shim(ctx, guardian_set_bump, vaa_body)
    }
}
