//! Utilities for the Core Bridge Program. These utilities are used to convert the legacy program to
//! use the Anchor framework.

use anchor_lang::prelude::*;

/// Trait for account schemas of legacy programs (intended for Core Bridge and Token Bridge, but can
/// be used for any legacy program). A legacy account requires a defined discriminator (if there is
/// none, yikes, then it will be an empty array) and a program ID, which will usually just be
/// `crate::ID` (defined using [declare_id](anchor_lang::prelude::declare_id)).
pub trait LegacyAccount<const N: usize>: AnchorSerialize + AnchorDeserialize + Clone {
    /// Account discriminator. If there is none, use an empty array (N == 0).
    const DISCRIMINATOR: [u8; N];

    /// Owner of the account.
    fn program_id() -> Pubkey;
}

/// Wrapper for legacy accounts implementing [LegacyAccount]. This wrapper provides the convenience
/// of not having to implement [AccountSerialize] and [AccountDeserialize] for each legacy account.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyAnchorized<const N: usize, T: LegacyAccount<N>>(T);

impl<const N: usize, T> AsRef<T> for LegacyAnchorized<N, T>
where
    T: LegacyAccount<N>,
{
    fn as_ref(&self) -> &T {
        &self.0
    }
}

impl<const N: usize, T> From<T> for LegacyAnchorized<N, T>
where
    T: LegacyAccount<N>,
{
    fn from(acct: T) -> Self {
        Self(acct)
    }
}

impl<const N: usize, T> Owner for LegacyAnchorized<N, T>
where
    T: LegacyAccount<N>,
{
    fn owner() -> Pubkey {
        T::program_id()
    }
}

impl<const N: usize, T> std::ops::Deref for LegacyAnchorized<N, T>
where
    T: LegacyAccount<N>,
{
    type Target = T;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl<const N: usize, T> std::ops::DerefMut for LegacyAnchorized<N, T>
where
    T: LegacyAccount<N>,
{
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl<const N: usize, T> AccountSerialize for LegacyAnchorized<N, T>
where
    T: LegacyAccount<N>,
{
    fn try_serialize<W: std::io::Write>(&self, writer: &mut W) -> Result<()> {
        if writer.write_all(&T::DISCRIMINATOR).is_err() {
            return err!(ErrorCode::AccountDidNotSerialize);
        }

        if AnchorSerialize::serialize(&self.0, writer).is_err() {
            return err!(ErrorCode::AccountDidNotSerialize);
        }
        Ok(())
    }
}

impl<const N: usize, T> AccountDeserialize for LegacyAnchorized<N, T>
where
    T: LegacyAccount<N>,
{
    fn try_deserialize(buf: &mut &[u8]) -> Result<Self> {
        if buf.len() < N {
            return err!(ErrorCode::AccountDidNotDeserialize);
        };
        let given_disc = &buf[..N];
        if T::DISCRIMINATOR != *given_disc {
            return err!(ErrorCode::AccountDidNotDeserialize);
        }
        Self::try_deserialize_unchecked(buf)
    }

    fn try_deserialize_unchecked(buf: &mut &[u8]) -> Result<Self> {
        let mut data: &[u8] = &buf[N..];
        Ok(Self(T::deserialize(&mut data)?))
    }
}

/// Trait used for legacy instruction handlers. It is used to process instructions from
/// legacy programs, where an enum defines the instruction type (one byte selector).
pub trait ProcessLegacyInstruction<'info, T: AnchorDeserialize>:
    Accounts<'info> + AccountsExit<'info> + ToAccountInfos<'info>
{
    /// This name is what gets written to in a program log similar to how Anchor instructions are
    /// logged. This name is logged in the process instruction method.
    const LOG_IX_NAME: &'static str;

    /// This function resembles an instruction handler method written for an ordinary Anchor
    /// program. This method gets invoked in the process instruction method.
    const ANCHOR_IX_FN: fn(Context<Self>, T) -> Result<()>;

    /// This method implements the same procedure Anchor performs in its codegen, where it creates
    /// a Context using the account context and invokes the instruction handler with the handler's
    /// arguments. It then performs clean up at the end by writing the account data back into the
    /// borrowed account data via exit.
    fn process_instruction(
        program_id: &Pubkey,
        mut account_infos: &[AccountInfo<'info>],
        mut ix_data: &[u8],
    ) -> Result<()> {
        #[cfg(not(feature = "no-log-ix-name"))]
        msg!("Instruction: {}", Self::LOG_IX_NAME);

        let mut bumps = std::collections::BTreeMap::new();

        // Generate accounts struct. This checks account constraints, including PDAs.
        let mut accounts = Self::try_accounts(
            program_id,
            &mut account_infos,
            ix_data,
            &mut bumps,
            &mut std::collections::BTreeSet::new(),
        )?;

        // Create new context of these accounts.
        let ctx = Context::new(program_id, &mut accounts, &[], bumps);

        // Execute method that takes this context with specified instruction arguments.
        Self::ANCHOR_IX_FN(ctx, T::deserialize(&mut ix_data)?)?;

        // Finally clean up (this sets data from account struct members into account data).
        accounts.exit(program_id)
    }
}
