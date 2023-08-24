use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, Config, GuardianSet},
    types::Timestamp,
    zero_copy::PostedVaaV1,
};
use anchor_lang::prelude::*;
use wormhole_raw_vaas::core::CoreBridgeGovPayload;
use wormhole_solana_common::{utils, NewAccountSize, SeedPrefix};

#[derive(Accounts)]
pub struct GuardianSetUpdate<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, Config>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.message_hash().as_ref()
        ],
        bump
    )]
    posted_vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_address().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_chain().to_be_bytes().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.sequence().to_be_bytes().as_ref(),
        ],
        bump,
    )]
    claim: Account<'info, Claim>,

    #[account(
        mut,
        seeds = [GuardianSet::SEED_PREFIX, &config.guardian_set_index.to_be_bytes()],
        bump,
    )]
    curr_guardian_set: Account<'info, GuardianSet>,

    #[account(
        init,
        payer = payer,
        space = try_compute_size(&posted_vaa)?,
        seeds = [GuardianSet::SEED_PREFIX, &(curr_guardian_set.index + 1).to_be_bytes()],
        bump,
    )]
    new_guardian_set: Account<'info, GuardianSet>,

    system_program: Program<'info, System>,
}

/// Read account info data assuming we are reading a guardian set update governance decree to
/// determine how large the new guardian set account needs to be given how many guardians there are
/// in the new set.
///
/// NOTE: This step does not have to fail because if this VAA is not what we expect, the access
/// control following the account context checks will fail.
fn try_compute_size(posted_vaa: &AccountInfo<'_>) -> Result<usize> {
    let acc_data = posted_vaa.try_borrow_data()?;
    let vaa = PostedVaaV1::parse(&acc_data)?;
    let gov_payload = CoreBridgeGovPayload::parse(vaa.payload())
        .map(|msg| msg.decree())
        .map_err(|_| error!(CoreBridgeError::InvalidGovernanceVaa))?;

    gov_payload
        .guardian_set_update()
        .map(|decree| GuardianSet::compute_size(decree.num_guardians().into()))
        .ok_or(error!(CoreBridgeError::InvalidGovernanceAction))
}

impl<'info> GuardianSetUpdate<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let config = &ctx.accounts.config;
        let acc_data = ctx.accounts.posted_vaa.data.borrow();
        let gov_payload = super::require_valid_posted_governance_vaa(&acc_data, config)?;

        // Encoded guardian set must be the next value after the current guardian set index.
        //
        // NOTE: Because try_compute_size already determined whether this governance payload is a
        // guardian set update, we are safe to unwrap here.
        require_eq!(
            gov_payload.guardian_set_update().unwrap().new_index(),
            config.guardian_set_index + 1,
            CoreBridgeError::InvalidGuardianSetIndex
        );

        // Done.
        Ok(())
    }
}

#[access_control(GuardianSetUpdate::constraints(&ctx))]
pub fn guardian_set_update(ctx: Context<GuardianSetUpdate>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_data = ctx.accounts.posted_vaa.data.borrow();
    let vaa = PostedVaaV1::parse(&acc_data).unwrap();

    let gov_payload = CoreBridgeGovPayload::parse(vaa.payload()).unwrap().decree();
    let decree = gov_payload.guardian_set_update().unwrap();

    // Deserialize new guardian set.
    let keys: Vec<[u8; 20]> = (0..usize::from(decree.num_guardians()))
        .map(|i| decree.guardian_at(i))
        .collect();
    for (i, guardian) in keys.iter().take(keys.len() - 1).enumerate() {
        // We disallow guardian pubkeys that have zero address.
        require!(
            utils::is_nonzero_array(guardian),
            CoreBridgeError::GuardianZeroAddress
        );

        // Check if this pubkey is a duplicate of any others.
        for other in keys.iter().skip(i + 1) {
            require!(guardian != other, CoreBridgeError::DuplicateGuardianAddress);
        }
    }

    // Set new guardian set account fields.
    ctx.accounts.new_guardian_set.set_inner(GuardianSet {
        index: ctx.accounts.curr_guardian_set.index + 1,
        creation_time: vaa.timestamp(),
        keys,
        expiration_time: Default::default(),
    });

    // Set the new index on the config program data.
    let config = &mut ctx.accounts.config;
    config.guardian_set_index += 1;

    // Now set the expiration time for the current guardian.
    let now = Clock::get().map(Timestamp::from)?;
    ctx.accounts.curr_guardian_set.expiration_time = now.saturating_add(&config.guardian_set_ttl);

    // Done.
    Ok(())
}
