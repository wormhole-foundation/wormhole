use crate::{
    constants::UPGRADE_SEED_PREFIX,
    error::CoreBridgeError,
    state::{BridgeConfig, BridgeProgramData, FeeCollector, GuardianSet},
    ID,
};
use anchor_lang::prelude::*;
use solana_program::{bpf_loader_upgradeable, program::invoke};
use wormhole_solana_common::{utils, BpfLoaderUpgradeable, NewAccountSize, SeedPrefix};

const INDEX_ZERO: u32 = 0;

#[derive(Accounts)]
#[instruction(args: InitializeArgs)]
pub struct Initialize<'info> {
    /// This account should be the same payer that deployed the Core Bridge BPF.
    #[account(mut)]
    deployer: Signer<'info>,

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
        space = wtf_compute_size(args.initial_guardians.len()),
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

    /// CHECK: Before we initialize this program, we need to verify that the upgrade authority is
    /// set for this program. To do so, call the `SetAuthority` instruction via the
    /// BpfLoaderUpgradeable native program.
    #[account(
        seeds = [UPGRADE_SEED_PREFIX],
        bump,
    )]
    upgrade_authority: AccountInfo<'info>,

    /// CHECK: BPF Loader Upgradeable program needs to modify this program's data to change the
    /// upgrade authority. We check this PDA address just in case there is another program that this
    /// deployer has deployed.
    ///
    /// NOTE: Set upgrade authority is scary because any public key can be used to set as the
    /// authority.
    #[account(
        mut,
        seeds = [ID.as_ref()],
        bump,
        seeds::program = bpf_loader_upgradeable_program,
    )]
    program_data: AccountInfo<'info>,

    system_program: Program<'info, System>,
    bpf_loader_upgradeable_program: Program<'info, BpfLoaderUpgradeable>,
}

fn wtf_compute_size(um: usize) -> usize {
    msg!("um... {}", um);
    GuardianSet::compute_size(um)
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct InitializeArgs {
    pub guardian_set_ttl_seconds: u32,
    pub fee_lamports: u64,
    pub initial_guardians: Vec<[u8; 20]>,
}

pub(crate) fn initialize(ctx: Context<Initialize>, args: InitializeArgs) -> Result<()> {
    let InitializeArgs {
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

    // Finally set the upgrade authority to this program's PDA address.
    invoke(
        &bpf_loader_upgradeable::set_upgrade_authority(
            &ID,
            &ctx.accounts.deployer.key(),
            Some(&ctx.accounts.upgrade_authority.key()),
        ),
        &ctx.accounts.to_account_infos(),
    )
    .map_err(Into::into)
}
