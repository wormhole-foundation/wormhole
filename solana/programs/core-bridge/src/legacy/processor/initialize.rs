use crate::{
    error::CoreBridgeError,
    legacy::instruction::LegacyInitializeArgs,
    state::{BridgeConfig, BridgeProgramData, FeeCollector, GuardianSet},
};
use anchor_lang::prelude::*;
use wormhole_solana_common::{utils, NewAccountSize, SeedPrefix};

const INDEX_ZERO: u32 = 0;

#[derive(Accounts)]
#[instruction(args: LegacyInitializeArgs)]
pub struct Initialize<'info> {
    /// Core Bridge data and config. This account is necessary to publish Wormhole messages and
    /// redeem governance VAAs.
    #[account(
        init,
        payer = deployer,
        space = BridgeProgramData::INIT_SPACE,
        seeds = [BridgeProgramData::seed_prefix()],
        bump,
    )]
    bridge: Account<'info, BridgeProgramData>,

    /// New guardian set account, acting as the active guardian set.
    ///
    /// NOTE: There are other Core Bridge smart contracts that take an additional guardian set index
    /// parameter to initialize a present-day guardian set at initialization. But because the Core
    /// Bridge already exists on Solana's mainnet and devnet, we keep initialization assuming the
    /// initial guardian set is index 0.
    #[account(
        init,
        payer = deployer,
        space = GuardianSet::compute_size(args.initial_guardians.len()),
        seeds = [GuardianSet::seed_prefix(), &INDEX_ZERO.to_be_bytes()],
        bump,
    )]
    guardian_set: Account<'info, GuardianSet>,

    /// System account that collects lamports for `post_message`.
    #[account(
        init,
        payer = deployer,
        space = FeeCollector::INIT_SPACE,
        seeds = [FeeCollector::seed_prefix()],
        bump,
        owner = system_program.key(),
    )]
    fee_collector: Account<'info, FeeCollector>,

    /// This account should be the same payer that deployed the Core Bridge BPF. But there is no
    /// enforcement of this in this instruction handler.
    #[account(mut)]
    deployer: Signer<'info>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

pub(crate) fn initialize(ctx: Context<Initialize>, args: LegacyInitializeArgs) -> Result<()> {
    let LegacyInitializeArgs {
        guardian_set_ttl_seconds,
        fee_lamports,
        initial_guardians,
    } = args;

    // We need at least one guardian for the initial guardian set.
    require!(
        !initial_guardians.is_empty(),
        CoreBridgeError::ZeroGuardians
    );

    // Check initial guardians.
    let mut keys = Vec::with_capacity(initial_guardians.len());
    for &guardian in initial_guardians.iter() {
        // We disallow guardian pubkeys that have zero address.
        require!(
            utils::is_nonzero_array(&guardian),
            CoreBridgeError::GuardianZeroAddress
        );

        // Check if this pubkey is a duplicate of any already added.
        require!(
            !keys.iter().any(|key| *key == guardian),
            CoreBridgeError::DuplicateGuardianAddress
        );
        keys.push(guardian);
    }

    // Set Bridge data account fields.
    ctx.accounts.bridge.set_inner(BridgeProgramData {
        guardian_set_index: INDEX_ZERO,
        last_lamports: ctx.accounts.fee_collector.to_account_info().lamports(),
        config: BridgeConfig {
            guardian_set_ttl: guardian_set_ttl_seconds.into(),
            fee_lamports,
        },
    });

    // Set guardian set account fields.
    ctx.accounts.guardian_set.set_inner(GuardianSet {
        index: INDEX_ZERO,
        creation_time: Clock::get().map(Into::into)?,
        keys,
        expiration_time: Default::default(),
    });

    Ok(())
}
