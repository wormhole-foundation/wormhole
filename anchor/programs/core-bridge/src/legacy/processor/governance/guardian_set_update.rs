use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{BridgeProgramData, Claim, GuardianSet, VaaV1LegacyAccount},
    types::Timestamp,
    utils::PostedGovernanceVaaV1,
};
use anchor_lang::prelude::*;
use wormhole_solana_common::{utils, NewAccountSize, SeedPrefix};
use wormhole_vaas::payloads::gov::core_bridge::Decree;

#[derive(Accounts)]
pub struct GuardianSetUpdate<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        seeds = [BridgeProgramData::seed_prefix()],
        bump,
    )]
    bridge: Account<'info, BridgeProgramData>,

    #[account(
        seeds = [
            PostedGovernanceVaaV1::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedGovernanceVaaV1>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            posted_vaa.emitter_address.as_ref(),
            &posted_vaa.emitter_chain.to_be_bytes(),
            &posted_vaa.sequence.to_be_bytes()
        ],
        bump,
    )]
    claim: Account<'info, Claim>,

    #[account(
        mut,
        seeds = [GuardianSet::seed_prefix(), &bridge.guardian_set_index.to_be_bytes()],
        bump,
    )]
    curr_guardian_set: Account<'info, GuardianSet>,

    #[account(
        init,
        payer = payer,
        space = try_compute_size(&posted_vaa.payload.decree)?,
        seeds = [GuardianSet::seed_prefix(), &(curr_guardian_set.index + 1).to_be_bytes()],
        bump,
    )]
    new_guardian_set: Account<'info, GuardianSet>,

    system_program: Program<'info, System>,
}

fn try_compute_size(decree: &Decree) -> Result<usize> {
    if let Decree::GuardianSetUpdate(inner) = decree {
        Ok(GuardianSet::compute_size(inner.guardians.len()))
    } else {
        err!(CoreBridgeError::InvalidGovernanceAction)
    }
}

impl<'info> GuardianSetUpdate<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let bridge = &ctx.accounts.bridge;
        let decree =
            crate::utils::require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa, bridge)?;

        if let Decree::GuardianSetUpdate(inner) = decree {
            // Encoded guardian set must be the next value after the current guardian set index.
            require_eq!(inner.new_index, bridge.guardian_set_index + 1);

            let guardians = &inner.guardians;
            for (i, guardian) in guardians.iter().take(guardians.len() - 1).enumerate() {
                // We disallow guardian pubkeys that have zero address.
                require!(
                    utils::is_nonzero_array(guardian),
                    CoreBridgeError::GuardianZeroAddress
                );

                // Check if this pubkey is a duplicate of any others.
                for other in guardians.iter().skip(i + 1) {
                    require!(guardian != other, CoreBridgeError::DuplicateGuardianAddress);
                }
            }

            // Done.
            Ok(())
        } else {
            err!(CoreBridgeError::InvalidGovernanceAction)
        }
    }
}

#[access_control(GuardianSetUpdate::accounts(&ctx))]
pub fn guardian_set_update(ctx: Context<GuardianSetUpdate>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    // We know this is the only variant that can be present given access control.
    if let Decree::GuardianSetUpdate(inner) = &mut ctx.accounts.posted_vaa.payload.decree {
        let index = inner.new_index;
        let keys = {
            let guardians = std::mem::take(&mut inner.guardians);

            unsafe {
                let mut keys = std::mem::ManuallyDrop::new(guardians);
                Vec::from_raw_parts(
                    keys.as_mut_ptr() as *mut [u8; 20],
                    keys.len(),
                    keys.capacity(),
                )
            }
        };

        // Set new guardian set account fields.
        ctx.accounts.new_guardian_set.set_inner(GuardianSet {
            index,
            creation_time: Clock::get().map(Into::into)?,
            keys,
            expiration_time: Default::default(),
        });

        // Set the new index on the bridge program data.
        let bridge = &mut ctx.accounts.bridge;
        bridge.guardian_set_index = index;

        // Now set the expiration time for the current guardian.
        ctx.accounts.curr_guardian_set.expiration_time = Clock::get()
            .map(|clock| Timestamp::from(clock).saturating_add(&bridge.guardian_set_ttl))?;
    } else {
        unreachable!()
    }

    // Done.
    Ok(())
}
