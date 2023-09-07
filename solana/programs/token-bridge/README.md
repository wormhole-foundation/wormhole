# wormhole-token-bridge-solana

This package implements Wormhole's Token Bridge specification on Solana with some modifications (due
to the nature of how Solana works). The program itself is written using the [Anchor] framework.

## Example Integration (Inbound Transfer)

In order to bridge assets into Solana with a program integrating with Token Bridge, there are a
couple of traits that you the integrator will have to implement:

- `CompleteTransfer<'info>`
  - Ensures that all Token Bridge accounts for inbound transfers are included in your
    [account context].
- `CreateAccount<'info>`
  - Requires payer and System program account infos.

These traits are found in the [SDK] submodule of the Token Bridge program crate.

```rust,ignore
use wormhole_token_bridge_solana::sdk as token_bridge_sdk;
```

Your account context may resemble the following:

```rust,ignore
#[derive(Accounts)]
pub struct RedeemHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        associated_token::mint = mint,
        associated_token::authority = payer,
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// CHECK: Mint of our token account. This account is mutable in case the transferred asset is
    /// Token Bridge wrapped, which requires the Token Bridge program to mint to a token account.
    #[account(
        mut,
        owner = token::ID
    )]
    mint: AccountInfo<'info>,

    /// CHECK: This account acts as the signer for our Token Bridge complete transfer with payload.
    /// This PDA validates the redeemer address as this program's ID.
    #[account(
        seeds = [token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX],
        bump,
    )]
    redeemer_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This account warehouses the
    /// VAA of the token transfer from another chain.
    encoded_vaa: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    token_bridge_claim: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_registered_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    #[account(mut)]
    token_bridge_custody_token_account: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    token_bridge_custody_authority: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_wrapped_asset: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_mint_authority: Option<AccountInfo<'info>>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}
```

This account context must have all of the accounts required by the Token Bridge program in order to
transfer assets into Solana:

- `token_bridge_program`
- `token_program` (SPL Token program pubkey).
- `vaa` (VAA account, either an Encoded VAA account or legacy Posted VAA V1 account).
  - **NOTE: These VAA accounts are created using a parse and verify workflow, which is executed
    outside of your program. Please refer to the Wormhole JS SDK for more info.**
- `token_bridge_claim` (Account used to prevent replay attacks ensuring that each VAA is redeemed
  only once).
- `token_bridge_registered_emitter` (Account reflecting which Token Bridge smart contract emits
  Wormhole messages, seeds = \[emitter_chain\]).
- `dst_token_account` (where the assets will be bridged from).
- `mint` (SPL Mint, which should be the same mint of your token account).
- `token_bridge_redeemer_authority` (seeds: ["redeemer"])
  - **NOTE: Your program ID is the redeemer in this case and must match the encoded redeemer address
    in the transfer VAA.**
- `token_bridge_custody_token_account` (required for native assets, seeds: \[mint.key\]).
- `token_bridge_custody_authority` (required for native assets, seeds: ["custody_signer"]).
- `token_bridge_mint_authority` (required for wrapped assets, seeds: ["mint_signer"]).
- `token_bridge_wrapped_asset` (required for wrapped assets, seeds: ["meta", mint.key]).

**You are not required to re-derive these PDA addresses in your program's account context because
the Token Bridge program already does these derivations. Doing so is a waste of compute units.**

The traits above would be implemented by calling `to_account_info` on the appropriate accounts in
your context.

By making sure that the `token_bridge_program` account is the correct program, your context will use
the [Program] account wrapper with the `TokenBridge` type.

Because redeeming asset transfers requires creating account(s), the `CompleteTransfer` trait
requires the `CreateAccount` trait, which defines a `payer` account, who has the lamports to send to
a new account, and the `system_program`, which is used via CPI to create accounts.

```rust,ignore
impl<'info> token_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for RedeemHelloWorld<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}
```

Finally implement the `CompleteTransfer` trait by providing the necessary Token Bridge accounts.

**NOTE: For transfers where the redeemer address is your program ID, the
`token_bridge_redeemer_authority` in this case is `Some(redeemer_authority)`, which is your
program's PDA address derived using `[b"redeemer"]` as its seeds. This seed prefix is provided for
you as `PROGRAM_REDEEMER_SEED_PREFIX` and is used in your account context to validate that the
correct redeemer authority is provided.**

```rust,ignore
impl<'info> token_bridge_sdk::cpi::CompleteTransfer<'info> for RedeemHelloWorld<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn vaa(&self) -> AccountInfo<'info> {
        self.encoded_vaa.to_account_info()
    }

    fn token_bridge_claim(&self) -> AccountInfo<'info> {
        self.token_bridge_claim.to_account_info()
    }

    fn token_bridge_registered_emitter(&self) -> AccountInfo<'info> {
        self.token_bridge_registered_emitter.to_account_info()
    }

    fn dst_token_account(&self) -> AccountInfo<'info> {
        self.payer_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.mint.to_account_info()
    }

    fn redeemer_authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.redeemer_authority.to_account_info())
    }

    fn token_bridge_custody_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_authority.clone()
    }

    fn token_bridge_custody_token_account(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_token_account.clone()
    }

    fn token_bridge_mint_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_mint_authority.clone()
    }

    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_wrapped_asset.clone()
    }
}
```

In your instruction handler/processor method, you would use the `complete_transfer` method from the
CPI SDK. The Token Bridge program will verify that your redeemer authority can be derived the same
way using the encoded redeemer address in your transfer VAA as your program ID (this validates the
correct redeemer address is used to redeem your transfer).

This method will invoke the Token Bridge to bridge assets in, which basically uses the VAA to show
proof of a transfer originating from another network (whose observation was made by the Guardians).

```rust,ignore
pub fn redeem_hello_world(ctx: Context<RedeemHelloWorld>) -> Result<()> {
    token_bridge_sdk::cpi::complete_transfer(
        ctx.accounts,
        Some(&[&[
            token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX,
            &[ctx.bumps["redeemer_authority"]],
        ]]),
    )
}
```

And that is all you need to do to transfer assets into Solana.

## Example Integration (Outbound Transfer)

In order to bridge assets from Solana with a program integrating with Token Bridge, there are a few
traits that you the integrator will have to implement:

- `TransferTokens<'info>`
  - Ensures that all Token Bridge accounts for outbound transfers are included in your
    [account context].
- `PublishMessage<'info>`
  - Ensures that all Core Bridge accounts are included in your [account context].
  - **NOTE: This includes having to implement `CreateAccount<'info>`. See
    [Core Bridge program documentation] for more details.**

These traits are found in the [SDK] submodule of the Token Bridge program crate.

```rust,ignore
use wormhole_token_bridge_solana::sdk::{self as token_bridge_sdk, core_bridge_sdk};
```

Your account context may resemble the following:

```rust,ignore

#[derive(Accounts)]
pub struct TransferHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        associated_token::mint = mint,
        associated_token::authority = payer,
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// CHECK: Mint of our token account.
    #[account(owner = token::ID)]
    mint: AccountInfo<'info>,

    /// CHECK: This account acts as the signer for our Token Bridge transfer with payload. This PDA
    /// validates the sender address as this program's ID.
    #[account(
        seeds = [token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX],
        bump,
    )]
    sender_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_transfer_authority: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    token_bridge_custody_token_account: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    token_bridge_custody_authority: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_wrapped_asset: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_core_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account will be created using a generated keypair.
    #[account(mut)]
    core_message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_fee_collector: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_program: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}
```

This account context must have all of the accounts required by the Token Bridge program in order to
transfer assets out:

- `token_bridge_program`
- `token_program` (SPL Token program pubkey).
- `src_token_account` (where the assets will be bridged from).
- `mint` (SPL Mint, which should be the same mint of your token account).
- `token_bridge_sender_authority` (seeds: ["sender"])
  - **NOTE: Your program ID is the sender in this case.**
- `token_bridge_transfer_authority` (seeds: ["authority_signer"]).
- `token_bridge_custody_token_account` (required for native assets, seeds: [mint.key]).
- `token_bridge_custody_authority` (required for native assets, seeds: ["custody_signer"]).
- `token_bridge_wrapped_asset` (required for wrapped assets, seeds: ["meta", mint.key]).

**You are not required to re-derive these PDA addresses in your program's account context because
the Token Bridge program already does these derivations. Doing so is a waste of compute units.**

The traits above would be implemented by calling `to_account_info` on the appropriate accounts in
your context.

By making sure that the `token_bridge_program` account is the correct program, your context will use
the [Program] account wrapper with the `TokenBridge` type.

Because transferring assets out message requires publishing a Wormhole message, you must implement
the `PublishMessage` trait and the other traits it depends on (`CreateAccount`). Please see the
[Core Bridge program documentation] for more details.

Implement the `PublishMessage` trait by providing the necessary Core Bridge accounts.

```rust,ignore
impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for TransferHelloWorld<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.token_bridge_core_emitter.to_account_info()
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        Some(self.core_fee_collector.to_account_info())
    }
}
```

And finally implement the `TransferTokens` trait by providing the necessary Token Bridge accounts.

**NOTE: For transfers where the sender address is your program ID, the
`token_bridge_sender_authority` in this case is `Some(sender_authority)`, which is your program's
PDA address derived using `[b"sender"]` as its seeds. This seed prefix is provided for you as
`PROGRAM_SENDER_SEED_PREFIX` and is used in your account context to validate that the correct sender
authority is provided.**

```rust,ignore
impl<'info> token_bridge_sdk::cpi::TransferTokens<'info> for TransferHelloWorld<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn src_token_account(&self) -> AccountInfo<'info> {
        self.payer_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.mint.to_account_info()
    }

    fn token_bridge_sender_authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.sender_authority.to_account_info())
    }

    fn token_bridge_transfer_authority(&self) -> AccountInfo<'info> {
        self.token_bridge_transfer_authority.to_account_info()
    }

    fn token_bridge_custody_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_authority.clone()
    }

    fn token_bridge_custody_token_account(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_token_account.clone()
    }

    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_wrapped_asset.clone()
    }
}
```

In your instruction handler/processor method, you would use the `transfer_tokens` method from the
CPI SDK with the `TransferTokensDirective::ProgramTransferWithPayload` with your program ID. The
Token Bridge program will verify that your sender authority can be derived the same way using the
provided program ID (this validates the correct sender address will be used for your transfer).

This directive with the other transfer arguments (`nonce`, `amount`, `redeemer`, `redeemer_chain`
and message `payload`) will invoke the Token Bridge to bridge assets out, which is basically a
Worhole message emitted by the Token Bridge observed by the Guardians. When the Wormhole Guardians
sign this message attesting to its observation, you may redeem this attested transfer (VAA) on the
specified redeemer's network (specified by redeemer_chain) where a Token Bridge smart contract is
deployed.

```rust,ignore
pub fn transfer_hello_world(ctx: Context<TransferHelloWorld>, amount: u64) -> Result<()> {
    let nonce = 420;
    let redeemer = [
        0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xde, 0xad, 0xbe, 0xef,
        0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad,
        0xbe, 0xef,
    ];
    let redeemer_chain = 2;
    let payload = b"Hello, world!".to_vec();

    token_bridge_sdk::cpi::transfer_tokens(
        ctx.accounts,
        token_bridge_sdk::cpi::TransferTokensDirective::ProgramTransferWithPayload {
            program_id: crate::ID,
            nonce,
            amount,
            redeemer,
            redeemer_chain,
            payload,
        },
        Some(&[&[
            token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX,
            &[ctx.bumps["sender_authority"]],
        ]]),
    )
}
```

And that is all you need to do to transfer assets from Solana.

## Putting it All Together

```rust,ignore
#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;
use anchor_spl::token;
use wormhole_token_bridge_solana::sdk::{self as token_bridge_sdk, core_bridge_sdk};

declare_id!("TokenBridgeHe11oWor1d1111111111111111111111");

#[derive(Accounts)]
pub struct RedeemHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        associated_token::mint = mint,
        associated_token::authority = payer,
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// CHECK: Mint of our token account. This account is mutable in case the transferred asset is
    /// Token Bridge wrapped, which requires the Token Bridge program to mint to a token account.
    #[account(
        mut,
        owner = token::ID
    )]
    mint: AccountInfo<'info>,

    /// CHECK: This account acts as the signer for our Token Bridge complete transfer with payload.
    /// This PDA validates the redeemer address as this program's ID.
    #[account(
        seeds = [token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX],
        bump,
    )]
    redeemer_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This account warehouses the
    /// VAA of the token transfer from another chain.
    encoded_vaa: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    token_bridge_claim: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_registered_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    #[account(mut)]
    token_bridge_custody_token_account: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    token_bridge_custody_authority: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_wrapped_asset: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_mint_authority: Option<AccountInfo<'info>>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> token_bridge_sdk::cpi::CreateAccount<'info> for RedeemHelloWorld<'info> {
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> token_bridge_sdk::cpi::CompleteTransfer<'info> for RedeemHelloWorld<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn vaa(&self) -> AccountInfo<'info> {
        self.encoded_vaa.to_account_info()
    }

    fn token_bridge_claim(&self) -> AccountInfo<'info> {
        self.token_bridge_claim.to_account_info()
    }

    fn token_bridge_registered_emitter(&self) -> AccountInfo<'info> {
        self.token_bridge_registered_emitter.to_account_info()
    }

    fn dst_token_account(&self) -> AccountInfo<'info> {
        self.payer_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.mint.to_account_info()
    }

    fn redeemer_authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.redeemer_authority.to_account_info())
    }

    fn token_bridge_custody_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_authority.clone()
    }

    fn token_bridge_custody_token_account(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_token_account.clone()
    }

    fn token_bridge_mint_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_mint_authority.clone()
    }

    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_wrapped_asset.clone()
    }

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }
}

#[derive(Accounts)]
pub struct TransferHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        associated_token::mint = mint,
        associated_token::authority = payer,
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// CHECK: Mint of our token account. This account is mutable in case the transferred asset is
    /// Token Bridge wrapped, which requires the Token Bridge program to burn from a token
    /// account.
    #[account(
        mut,
        owner = token::ID
    )]
    mint: AccountInfo<'info>,

    /// CHECK: This account acts as the signer for our Token Bridge transfer with payload. This PDA
    /// validates the sender address as this program's ID.
    #[account(
        seeds = [token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX],
        bump,
    )]
    sender_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_transfer_authority: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    #[account(mut)]
    token_bridge_custody_token_account: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    token_bridge_custody_authority: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_wrapped_asset: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_core_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account will be created using a generated keypair.
    #[account(mut)]
    core_message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_fee_collector: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_program: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> token_bridge_sdk::cpi::CreateAccount<'info> for TransferHelloWorld<'info> {
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for TransferHelloWorld<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.token_bridge_core_emitter.to_account_info()
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        Some(self.core_fee_collector.to_account_info())
    }
}

impl<'info> token_bridge_sdk::cpi::TransferTokens<'info> for TransferHelloWorld<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn core_message(&self) -> AccountInfo<'info> {
        self.core_message.to_account_info()
    }

    fn src_token_account(&self) -> AccountInfo<'info> {
        self.payer_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.mint.to_account_info()
    }

    fn sender_authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.sender_authority.to_account_info())
    }

    fn token_bridge_transfer_authority(&self) -> AccountInfo<'info> {
        self.token_bridge_transfer_authority.to_account_info()
    }

    fn token_bridge_custody_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_authority.clone()
    }

    fn token_bridge_custody_token_account(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_token_account.clone()
    }

    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_wrapped_asset.clone()
    }
}

#[program]
pub mod token_bridge_hello_world {
    use super::*;

    pub fn redeem_hello_world(ctx: Context<RedeemHelloWorld>) -> Result<()> {
        token_bridge_sdk::cpi::complete_transfer(
            ctx.accounts,
            Some(&[&[
                token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX,
                &[ctx.bumps["redeemer_authority"]],
            ]]),
        )
    }

    pub fn transfer_hello_world(ctx: Context<TransferHelloWorld>, amount: u64) -> Result<()> {
        let nonce = 420;
        let redeemer = [
            0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xde, 0xad, 0xbe, 0xef,
            0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad,
            0xbe, 0xef,
        ];
        let redeemer_chain = 2;
        let payload = b"Hello, world!".to_vec();

        token_bridge_sdk::cpi::transfer_tokens(
            ctx.accounts,
            token_bridge_sdk::cpi::TransferTokensDirective::ProgramTransferWithPayload {
                program_id: crate::ID,
                nonce,
                amount,
                redeemer,
                redeemer_chain,
                payload,
            },
            Some(&[&[
                token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX,
                &[ctx.bumps["sender_authority"]],
            ]]),
        )
    }
}

```

[account context]: https://docs.rs/anchor-lang/latest/anchor_lang/derive.Accounts.html
[anchor]: https://docs.rs/anchor-lang/latest/anchor_lang/
[core bridge program documentation]: https://docs.rs/wormhole-core-bridge-solana
[program]: https://docs.rs/anchor-lang/latest/anchor_lang/accounts/program/struct.Program.html
[sdk]: https://docs.rs/wormhole-token-bridge-solana/latest/wormhole_token_bridge_solana/sdk/cpi/index.html
