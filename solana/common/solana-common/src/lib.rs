#![allow(clippy::result_large_err)]

mod native_program;
pub use native_program::*;

pub mod utils;

pub use wormhole_attribute_legacy_account::legacy_account;

use anchor_lang::{
    prelude::{AnchorDeserialize, ErrorCode, Owner, Pubkey},
    solana_program::entrypoint::MAX_PERMITTED_DATA_INCREASE,
};
use anyhow::{anyhow, ensure};

pub trait LegacyDiscriminator<const N: usize> {
    const LEGACY_DISCRIMINATOR: [u8; N];

    fn require_discriminator(acct_data: &mut &[u8]) -> anchor_lang::Result<()>
    where
        [u8; N]: AnchorDeserialize,
    {
        utils::require_discriminator(acct_data, Self::LEGACY_DISCRIMINATOR)
    }
}

pub trait SeedPrefix {
    /// Get the arbitrary prefix of this account's PDA address seeds.
    fn seed_prefix() -> &'static [u8];
}

pub trait NewAccountSize {
    /// With this method's argument value, compute how large the account must be to satisfy this
    /// constraint.
    fn compute_size(value: usize) -> usize;

    /// This method attempts to convert a given value to `usize` and execute `compute_size`. This
    /// method will succeed if the computed size is less than `MAX_PERMITTED_DATA_INCREASE`, which
    /// at this time is 10KB (10,240 bytes).
    fn try_compute_size<T>(value: T) -> anyhow::Result<usize>
    where
        usize: TryFrom<T>,
    {
        let specified = usize::try_from(value).map_err(|_| anyhow!("cannot usize::try_from"))?; //CoreBridgeError::InvalidDataConversion)?;
        let size = Self::compute_size(specified);
        ensure!(size <= MAX_PERMITTED_DATA_INCREASE, "Realloc Exceeds Limit");

        Ok(size)
    }

    /// Similar to `try_compute_size`, but instead caps the calculated size to
    /// `MAX_PERMITTED_DATA_INCREASE`, which at this time is 10KB (10,240 bytes).
    fn compute_size_to_max(specified: usize) -> usize {
        let size = Self::compute_size(specified);

        if size > MAX_PERMITTED_DATA_INCREASE {
            MAX_PERMITTED_DATA_INCREASE
        } else {
            size
        }
    }
}

pub trait RequireAuthority {
    /// Get the `Pubkey` of this struct's authority.
    fn authority_key(&self) -> Pubkey;

    /// Assign this struct's authority.
    fn set_authority(&mut self, authority: &Pubkey) -> &mut Self;
}

pub trait AccountBump {
    /// Get the value of this struct's bump seed.
    fn bump_seed(&self) -> u8;

    /// Assign the value of a bump seed to this struct.
    fn set_bump_seed(&mut self, bump: u8);

    /// Seeds as a vector of byte slices.
    ///
    /// NOTE: This method cannot be used as a part of the `account` macro when an instruction's
    /// account context uses `#[derive(Accounts)]` because that macro parses an arbitrary array of
    /// byte slices.
    fn seeds(&self) -> Vec<&[u8]>;

    /// This method returns the same values as `Pubkey::create_program_address`, but uses `seeds` to
    /// derive PDA address with its own bump seed.
    fn create_program_address(&self) -> anchor_lang::Result<Pubkey>
    where
        Self: Owner,
    {
        let mut seeds = self.seeds();
        let bump_seed = &[self.bump_seed()];
        seeds.push(bump_seed);
        Pubkey::create_program_address(&seeds, &Self::owner())
            .map_err(|_| ErrorCode::ConstraintSeeds.into())
    }

    /// This method returns the same values as `Pubkey::find_program_address`, but uses `seeds` to
    /// derive PDA address and bump seed.
    fn find_program_address(&self) -> (Pubkey, u8)
    where
        Self: Owner,
    {
        Pubkey::find_program_address(&self.seeds(), &Self::owner())
    }
}

/// This macro resembles the codegen that exists in generating the entrypoint for a Solana program
/// written in Anchor.
///
/// Legacy instructions now leverage Anchor's account checking by creating account contexts similar
/// to how Solitaire used to work. The benefit to using Anchor's account contexts is more checking
/// happens upfront (i.e. before the instruction is executed) so that the processor can fail faster
/// in case of an error (e.g. validating deserialized account data against another account). This
/// macro also leverages various Anchor handlers like creating accounts within an account context,
/// which cleans up business logic in the instruction methods.
///
/// Because Anchor's codegen macros auto-magically write account data after execution, this macro
/// performs the same operation after executing an instruction method.
#[macro_export]
macro_rules! process_anchorized_legacy_instruction {
    (
        $program_id: expr,
        $instruction_name: expr,
        $account_type: ty,
        $account_infos: expr,
        $ix_data: expr,
        $execute: expr,
        $args: expr
    ) => {{
        {
            #[cfg(not(feature = "no-log-ix-name"))]
            msg!("Instruction: {}", $instruction_name);

            let mut bumps = std::collections::BTreeMap::new();

            // Generate accounts struct. This checks account constraints, including PDAs.
            let mut accounts = <$account_type>::try_accounts(
                &$program_id,
                &mut $account_infos,
                &$ix_data[1..],
                &mut bumps,
                &mut std::collections::BTreeSet::new(),
            )?;

            // Create new context of these accounts.
            let ctx = Context::new(&$program_id, &mut accounts, &[], bumps);

            // Execute method that takes this context with specified instruction arguments.
            $execute(ctx, $args)?;

            // Finally clean up (this sets data from account struct members into account data).
            accounts.exit(&$program_id)
        }
    }};
}
