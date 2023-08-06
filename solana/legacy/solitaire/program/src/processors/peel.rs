//! Peeling.
//!
//! The accounts in Solitaire programs are defined via layers of types, when each layer is peeled
//! off it performs checks, parsing, and any other desired side-effect. The mechanism for this is
//! the peel trait, which defines a set of types that recursively construct the desired type.

use borsh::BorshDeserialize;
use solana_program::{
    pubkey::Pubkey,
    sysvar::Sysvar as SolanaSysvar,
};

use crate::{
    processors::seeded::{
        AccountOwner,
        Owned,
    },
    trace,
    types::*,
    Context,
    Result,
    SolitaireError,
};
use borsh::BorshSerialize;

/// Generic Peel trait. This provides a way to describe what each "peeled"
/// layer of our constraints should check.
pub trait Peel<'a, 'b: 'a> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self>
    where
        Self: Sized;

    fn persist(&self, program_id: &Pubkey) -> Result<()>;
}

/// Peel a nullable value (0-account means None)
impl<'a, 'b: 'a, T: Peel<'a, 'b>> Peel<'a, 'b> for Option<T> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        // Check for 0-account
        if ctx.info.key == &Pubkey::new_from_array([0u8; 32]) {
            trace!(&format!(
                "Peeled {} is None, returning",
                std::any::type_name::<Option<T>>()
            ));
            Ok(None)
        } else {
            Ok(Some(T::peel(ctx)?))
        }
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        if let Some(s) = self.as_ref() {
            T::persist(s, program_id)
        } else {
            trace!(&format!(
                "Peeled {} is None, not persisting",
                std::any::type_name::<Option<T>>()
            ));
            Ok(())
        }
    }
}

/// Peel a Derived Key
impl<'a, 'b: 'a, T: Peel<'a, 'b>, const SEED: &'static str> Peel<'a, 'b> for Derive<T, SEED> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        // Attempt to Derive SEED
        let (derived, _bump) = Pubkey::find_program_address(&[SEED.as_ref()], ctx.this);
        match derived == *ctx.info.key {
            true => T::peel(ctx).map(|v| Derive(v)),
            _ => Err(SolitaireError::InvalidDerive(*ctx.info.key, derived)),
        }
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }
}

/// Peel a Mutable key.
impl<'a, 'b: 'a, T: Peel<'a, 'b>> Peel<'a, 'b> for Mut<T> {
    fn peel<I>(mut ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        ctx.immutable = false;
        match ctx.info.is_writable {
            true => T::peel(ctx).map(|v| Mut(v)),
            _ => Err(SolitaireError::InvalidMutability(
                *ctx.info.key,
                ctx.info.is_writable,
            )),
        }
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }
}

impl<'a, 'b: 'a, T: Peel<'a, 'b>> Peel<'a, 'b> for MaybeMut<T> {
    fn peel<I>(mut ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        ctx.immutable = !ctx.info.is_writable;
        T::peel(ctx).map(|v| MaybeMut(v))
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }
}

/// Peel a Signer.
impl<'a, 'b: 'a, T: Peel<'a, 'b>> Peel<'a, 'b> for Signer<T> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        match ctx.info.is_signer {
            true => T::peel(ctx).map(|v| Signer(v)),
            _ => Err(SolitaireError::InvalidSigner(*ctx.info.key)),
        }
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }
}

/// Expicitly depend upon the System account.
impl<'a, 'b: 'a, T: Peel<'a, 'b>> Peel<'a, 'b> for System<T> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        match true {
            true => T::peel(ctx).map(|v| System(v)),
            _ => panic!(),
        }
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        T::persist(self, program_id)
    }
}

/// Peel a Sysvar
impl<'a, 'b: 'a, Var> Peel<'a, 'b> for Sysvar<'b, Var>
where
    Var: SolanaSysvar,
{
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        match Var::check_id(ctx.info.key) {
            true => Ok(Sysvar(ctx.info.clone(), Var::from_account_info(ctx.info)?)),
            _ => Err(SolitaireError::InvalidSysvar(*ctx.info.key)),
        }
    }

    fn persist(&self, _program_id: &Pubkey) -> Result<()> {
        Ok(())
    }
}

/// This is our structural recursion base case, the trait system will stop generating new nested
/// calls here.
impl<'a, 'b: 'a> Peel<'a, 'b> for Info<'b> {
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        if ctx.immutable && ctx.info.is_writable {
            return Err(SolitaireError::InvalidMutability(
                *ctx.info.key,
                ctx.info.is_writable,
            ));
        }

        Ok(ctx.info.clone())
    }

    fn persist(&self, _program_id: &Pubkey) -> Result<()> {
        Ok(())
    }
}

/// This is our structural recursion base case, the trait system will stop generating new nested
/// calls here.
impl<
        'a,
        'b: 'a,
        T: BorshDeserialize + BorshSerialize + Owned + Default,
        const IS_INITIALIZED: AccountState,
    > Peel<'a, 'b> for Data<'b, T, IS_INITIALIZED>
{
    fn peel<I>(ctx: &mut Context<'a, 'b, I>) -> Result<Self> {
        if ctx.immutable && ctx.info.is_writable {
            return Err(SolitaireError::InvalidMutability(
                *ctx.info.key,
                ctx.info.is_writable,
            ));
        }

        // If we're initializing the type, we should emit system/rent as deps.
        let (initialized, data): (bool, T) = match IS_INITIALIZED {
            AccountState::Uninitialized => {
                if !ctx.info.data.borrow().is_empty() {
                    return Err(SolitaireError::AlreadyInitialized(*ctx.info.key));
                }
                (false, T::default())
            }
            AccountState::Initialized => (true, T::try_from_slice(*ctx.info.data.borrow_mut())?),
            AccountState::MaybeInitialized => {
                if ctx.info.data.borrow().is_empty() {
                    (false, T::default())
                } else {
                    (true, T::try_from_slice(*ctx.info.data.borrow_mut())?)
                }
            }
        };

        if initialized {
            match data.owner() {
                AccountOwner::This => {
                    if ctx.info.owner != ctx.this {
                        return Err(SolitaireError::InvalidOwner(*ctx.info.owner));
                    }
                }
                AccountOwner::Other(v) => {
                    if *ctx.info.owner != v {
                        return Err(SolitaireError::InvalidOwner(*ctx.info.owner));
                    }
                }
                AccountOwner::Any => {}
            };
        }

        Ok(Data(Box::new(ctx.info.clone()), data))
    }

    fn persist(&self, program_id: &Pubkey) -> Result<()> {
        // TODO: Introduce Mut<> to solve the check we really want to make here.
        if self.0.owner != program_id {
            return Ok(());
        }

        // It is also a malformed program to attempt to write to a non-writeable account.
        if !self.0.is_writable {
            return Ok(());
        }

        self.1.serialize(&mut *self.0.data.borrow_mut())?;

        Ok(())
    }
}
