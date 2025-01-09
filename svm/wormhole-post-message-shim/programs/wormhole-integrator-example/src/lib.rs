use anchor_lang::prelude::*;

declare_id!("AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ");

mod instructions;
pub(crate) use instructions::*;

#[program]
pub mod wormhole_integrator_example {
    use super::*;

    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        instructions::initialize(ctx)
    }

    pub fn consume_vaa(ctx: Context<ConsumeVaa>, vaa_body: Vec<u8>) -> Result<()> {
        instructions::consume_vaa(ctx, vaa_body)
    }

    pub fn post_message(ctx: Context<PostMessage>) -> Result<()> {
        instructions::post_message(ctx)
    }
}
