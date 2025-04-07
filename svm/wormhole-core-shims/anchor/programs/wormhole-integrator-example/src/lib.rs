use anchor_lang::prelude::*;

declare_id!("AEwubmehHNvkMXoH2C5MgDSemZgQ3HUSYpeaF3UrNZdQ");

mod instructions;
pub(crate) use instructions::*;

#[program]
pub mod wormhole_integrator_example {
    use super::*;

    /// This example instruction posts an empty message during initialize in order to have one less
    /// CPI depth on subsequent posts.
    ///
    /// NOTE: this example does not replay protect the call. Typically, this may be done with an
    /// `init` constraint on an account that was used to store config that is also set up during initialization.
    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        instructions::initialize(ctx)
    }

    /// This example instruction takes the guardian signatures account previously posted to the verify vaa shim
    /// with `post_signatures` and the corresponding guardian set from the core bridge and verifies a
    /// provided VAA body against it. `close_signatures` on the shim should be called immediately
    /// afterwards in order to reclaim the rent lamports taken by `post_signatures`.
    pub fn consume_vaa(
        ctx: Context<ConsumeVaa>,
        guardian_set_bump: u8,
        vaa_body: Vec<u8>,
    ) -> Result<()> {
        instructions::consume_vaa(ctx, guardian_set_bump, vaa_body)
    }

    /// This example instruction posts a message via the post message shim.
    pub fn post_message(ctx: Context<PostMessage>) -> Result<()> {
        instructions::post_message(ctx)
    }
}
