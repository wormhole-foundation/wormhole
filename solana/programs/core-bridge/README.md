# wormhole-core-bridge-solana

This package implements Wormhole's Core Bridge specification on Solana with some modifications (due
to the nature of how Solana works). The program itself is written using the [Anchor] framework.

## Example Integration

In order to publish a Wormhole message from another program using its program ID as an emitter
address, there are a few traits that you the integrator will have to implement:

- `PublishMessage<'info>`
  - Ensures that all Core Bridge accounts are included in your [account context].
- `CreateAccount<'info>`
  - Requires payer and System program account infos.

These traits are found in the [SDK] submodule of the Core Bridge program crate.

```rust,ignore
use wormhole_core_bridge_solana::sdk as core_bridge_sdk;
```

Your account context may resemble the following:

```rust,ignore
#[derive(Accounts)]
pub struct PublishHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages as
    /// its program ID.
    #[account(
        seeds = [core_bridge_sdk::PROGRAM_EMITTER_SEED_PREFIX],
        bump,
    )]
    core_program_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account will be created by the Core Bridge program using a generated keypair.
    #[account(mut)]
    core_message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}
```

This account context must have all of the accounts required by the Core Bridge program in order to
publish a Wormhole message:

- `core_message` (which in this example is just a keypair generated off-chain).
- `core_emitter_authority` (seeds: ["emitter"]).
  - **NOTE: Your program ID is the emitter in this case, which requires these specific seeds.**
- `core_bridge_config` (seeds: ["Bridge"]).
- `core_emitter_sequence` (seeds: ["Sequence", your_program_id]).
  - **NOTE: Your program ID is the emitter in this case, which is why the emitter sequence PDA
    address is derived using this pubkey.**
- `core_fee_collector` (seeds ["fee_collector"]).

**You are not required to re-derive these PDA addresses in your program's account context because
the Core Bridge program already does these derivations. Doing so is a waste of compute units.**

The traits above would be implemented by calling `to_account_info` on the appropriate accounts in
your context.

By making sure that the `core_bridge_program` account is the correct program, your context will use
the [Program] account wrapper with the `CoreBridge` type.

Because publishing a Wormhole message requires creating account(s), the `PublishMessage` trait
requires the `CreateAccount` trait, which defines a `payer` account, who has the lamports to send to
a new account, and the `system_program`, which is used via CPI to create accounts.

```rust,ignore
impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for PublishHelloWorld<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}
```

Finally implement the `PublishMessage` trait by providing the necessary Core Bridge accounts.

**NOTE: For messages where the emitter address is your program ID, the `core_emitter_authority` is
your program's PDA address derived using `[b"emitter"]` as its seeds. This seed prefix is provided
for you as `PROGRAM_EMITTER_SEED_PREFIX` and is used in your account context to validate that the
correct emitter authority is provided.**

```rust,ignore
impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for PublishHelloWorld<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.core_program_emitter.to_account_info()
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        Some(self.core_fee_collector.to_account_info())
    }
}
```

In your instruction handler/processor method, you would use the `publish_message` method from the
CPI SDK with the `PublishMessageDirective::ProgramMessage` with your program ID. The Core Bridge
program will verify that your emitter authority can be derived the same way using the provided
program ID (this validates the correct emitter address will be used for your Wormhole message).

This directive with the other message arguments (`nonce`, Solana `commitment` and message `payload`)
will invoke the Core Bridge to create a message account observed by the Guardians. When the Wormhole
Guardians sign this message attesting to its observation, you may redeem this attested message (VAA)
on any network where a Core Bridge smart contract is deployed.

```rust,ignore
pub fn publish_hello_world(ctx: Context<PublishHelloWorld>) -> Result<()> {
    let nonce = 420;
    let payload = b"Hello, world!".to_vec();

    core_bridge_sdk::cpi::publish_message(
        ctx.accounts,
        &ctx.accounts.core_message,
        core_bridge_sdk::cpi::PublishMessageDirective::ProgramMessage {
            program_id: crate::ID,
            nonce,
            payload,
            commitment: core_bridge_sdk::types::Commitment::Finalized,
        },
        Some(&[&[
            core_bridge_sdk::PROGRAM_EMITTER_SEED_PREFIX,
            &[ctx.bumps["core_program_emitter"]],
        ]]),
    )
}
```

And that is all you need to do to emit a Wormhole message from Solana.

## Putting it All Together

```rust,ignore
#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;
use wormhole_core_bridge_solana::sdk as core_bridge_sdk;

declare_id!("CoreBridgeHe11oWor1d11111111111111111111111");

#[derive(Accounts)]
pub struct PublishHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages as
    /// its program ID.
    #[account(
        seeds = [core_bridge_sdk::PROGRAM_EMITTER_SEED_PREFIX],
        bump,
    )]
    core_program_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account will be created by the Core Bridge program using a generated keypair.
    #[account(mut)]
    core_message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for PublishHelloWorld<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for PublishHelloWorld<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.core_program_emitter.to_account_info()
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        Some(self.core_fee_collector.to_account_info())
    }
}

#[program]
pub mod core_bridge_hello_world {
    use super::*;

    pub fn publish_hello_world(ctx: Context<PublishHelloWorld>) -> Result<()> {
        let nonce = 420;
        let payload = b"Hello, world!".to_vec();

        core_bridge_sdk::cpi::publish_message(
            ctx.accounts,
            &ctx.accounts.core_message,
            core_bridge_sdk::cpi::PublishMessageDirective::ProgramMessage {
                program_id: crate::ID,
                nonce,
                payload,
                commitment: core_bridge_sdk::types::Commitment::Finalized,
            },
            Some(&[&[
                core_bridge_sdk::PROGRAM_EMITTER_SEED_PREFIX,
                &[ctx.bumps["core_program_emitter"]],
            ]]),
        )
    }
}
```

[account context]: https://docs.rs/anchor-lang/latest/anchor_lang/derive.Accounts.html
[anchor]: https://docs.rs/anchor-lang/latest/anchor_lang/
[program]: https://docs.rs/anchor-lang/latest/anchor_lang/accounts/program/struct.Program.html
[sdk]: https://docs.rs/wormhole-core-bridge-solana/latest/wormhole_core_bridge_solana/sdk/cpi/index.html
