#![feature(const_generics)]
#![feature(const_generics_defaults)]
#![allow(warnings)]

//! Client-specific code

pub use solana_program::pubkey::Pubkey;
use solana_program::sysvar::Sysvar as SolSysvar;
pub use solana_sdk;

pub use solana_sdk::{
    instruction::{
        AccountMeta,
        Instruction,
    },
    signature::{
        Keypair,
        Signer as SolSigner,
    },
};

use borsh::BorshSerialize;

use solitaire::{
    AccountState,
    CPICall,
    FromAccounts,
    Info,
    Mut,
    Sysvar,
};
pub use solitaire::{
    Data,
    Derive,
    Keyed,
    Owned,
    Signer,
};

type StdResult<T, E> = std::result::Result<T, E>;

pub type ErrBox = Box<dyn std::error::Error>;

/// The sum type for clearly specifying the accounts required on client side.
#[derive(Debug)]
pub enum AccEntry {
    /// Least privileged account.
    Unprivileged(Pubkey),
    /// Least privileged account, read-only.
    UnprivilegedRO(Pubkey),

    /// Accounts that need to sign a Solana call
    Signer(Keypair),
    /// Accounts that need to sign a Solana call, read-only.
    SignerRO(Keypair),

    /// Program addresses for unprivileged cross calls
    CPIProgram(Pubkey, Box<dyn ToInstruction>),
    /// Program addresses for privileged cross calls
    CPIProgramSigner(Keypair, Box<dyn ToInstruction>),

    /// Key decided from SPL constants
    Sysvar(Pubkey),

    /// Key derived from constants and/or program address
    Derived(Pubkey),
    /// Key derived from constants and/or program address, read-only.
    DerivedRO(Pubkey),
}

/// Types implementing Wrap are those that can be turned into a
/// partial account vector for a program call.
pub trait Wrap {
    fn wrap(_: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox>;

    /// If the implementor wants to sign using other AccEntry
    /// variants, they should override this. Multiple keypairs may be
    /// used in multi-account AccEntry variants.
    fn partial_signer_keypairs(a: &AccEntry) -> Vec<Keypair> {
        use AccEntry::*;
        match a {
            // A panic on unwrap below here would imply solana keypair code does not understand its own bytes
            Signer(pair) | SignerRO(pair) => {
                vec![Keypair::from_bytes(pair.to_bytes().to_vec().as_slice()).unwrap()]
            }
            _other => vec![],
        }
    }
}

impl<'a, 'b: 'a, T> Wrap for Signer<T>
where
    T: Keyed<'a, 'b>,
{
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        use AccEntry::*;
        match a {
            Signer(pair) => Ok(vec![AccountMeta::new(pair.pubkey(), true)]),
            SignerRO(pair) => Ok(vec![AccountMeta::new_readonly(pair.pubkey(), true)]),
            other => Err(format!(
                "{} must be passed as Signer or SignerRO",
                std::any::type_name::<Self>()
            )
            .into()),
        }
    }
}

impl<'a, 'b: 'a, T, const Seed: &'static str> Wrap for Derive<T, Seed> {
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        match a {
            AccEntry::Derived(program_id) => {
                let (k, extra_seed) = Pubkey::find_program_address(&[Seed.as_bytes()], &program_id);

                Ok(vec![AccountMeta::new(k, false)])
            }
            AccEntry::DerivedRO(program_id) => {
                let (k, extra_seed) = Pubkey::find_program_address(&[Seed.as_bytes()], &program_id);

                Ok(vec![AccountMeta::new_readonly(k, false)])
            }
            other => Err(format!(
                "{} must be passed as Derived or DerivedRO",
                std::any::type_name::<Self>()
            )
            .into()),
        }
    }
}

impl<'a, T, const IsInitialized: AccountState> Wrap for Data<'a, T, IsInitialized>
where
    T: BorshSerialize + Owned + Default,
{
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        use AccEntry::*;
        use AccountState::*;
        match IsInitialized {
            Initialized => match a {
                Unprivileged(k) => Ok(vec![AccountMeta::new(*k, false)]),
                UnprivilegedRO(k) => Ok(vec![AccountMeta::new_readonly(*k, false)]),
                Signer(pair) => Ok(vec![AccountMeta::new(pair.pubkey(), true)]),
                SignerRO(pair) => Ok(vec![AccountMeta::new_readonly(pair.pubkey(), true)]),
                _other => Err(format!("{} with IsInitialized = {:?} must be passed as Unprivileged, Signer or the respective read-only variant", std::any::type_name::<Self>(), a).into())
            },
            Uninitialized => match a {
                Unprivileged(k) => Ok(vec![AccountMeta::new(*k, false)]),
                Signer(pair) => Ok(vec![AccountMeta::new(pair.pubkey(), true)]),
                _other => Err(format!("{} with IsInitialized = {:?} must be passed as Unprivileged or Signer (write access required for initialization)", std::any::type_name::<Self>(), a).into())
            }
            MaybeInitialized => match a {
                Unprivileged(k) => Ok(vec![AccountMeta::new(*k, false)]),
                Signer(pair) => Ok(vec![AccountMeta::new(pair.pubkey(), true)]),
                _other => Err(format!("{} with IsInitialized = {:?} must be passed as Unprivileged or Signer (write access required in case of initialization)", std::any::type_name::<Self>(), a).into())
            }
        }
    }
}

impl<'b, Var> Wrap for Sysvar<'b, Var>
where
    Var: SolSysvar,
{
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        if let AccEntry::Sysvar(k) = a {
            if Var::check_id(k) {
                Ok(vec![AccountMeta::new_readonly(k.clone(), false)])
            } else {
                Err(format!(
                    "{} does not point at sysvar {}",
                    k,
                    std::any::type_name::<Var>()
                )
                .into())
            }
        } else {
            Err(format!("{} must be passed as Sysvar", std::any::type_name::<Self>()).into())
        }
    }
}

impl<'b> Wrap for Info<'b> {
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        match a {
            AccEntry::UnprivilegedRO(k) => Ok(vec![AccountMeta::new_readonly(k.clone(), false)]),
            AccEntry::Unprivileged(k) => Ok(vec![AccountMeta::new(k.clone(), false)]),
            _other => Err(format!(
                "{} must be passed as Unprivileged or UnprivilegedRO",
                std::any::type_name::<Self>()
            )
            .into()),
        }
    }
}

impl<'b, T: Wrap> Wrap for Mut<T> {
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        match a {
            AccEntry::Unprivileged(_) | AccEntry::Signer(_) | AccEntry::Derived(_) => {
                Ok(T::wrap(a)?)
            }
            _other => Err(format!(
                "{} must be passed as a mutable AccEntry, such as Unprivileged, Signer or Derived",
                std::any::type_name::<Self>()
            )
            .into()),
        }
    }
}

impl<'a, 'b: 'a, 'c, T> Wrap for CPICall<T> {
    fn wrap(a: &AccEntry) -> StdResult<Vec<AccountMeta>, ErrBox> {
        match a {
            AccEntry::CPIProgram(xprog_k, xprog_accs) => {
                let mut v = vec![AccountMeta::new_readonly(xprog_k.clone(), false)];
                v.append(&mut xprog_accs.acc_metas(&xprog_k)?);
                Ok(v)
            }
            AccEntry::CPIProgramSigner(xprog_pair, xprog_accs) => {
                let mut v = vec![AccountMeta::new_readonly(
                    xprog_pair.pubkey().clone(),
                    false,
                )];
                v.append(&mut xprog_accs.acc_metas(&xprog_pair.pubkey())?);
                Ok(v)
            }
            _other => Err(format!(
                "{} must be passed as CPIProgram or CPIProgramSigner",
                std::any::type_name::<Self>()
            )
            .into()),
        }
    }

    fn partial_signer_keypairs(a: &AccEntry) -> Vec<Keypair> {
        match a {
            AccEntry::CPIProgram(xprog_k, xprog_accs) => xprog_accs.signer_keypairs(),
            AccEntry::CPIProgramSigner(xprog_pair, xprog_accs) => {
                let mut v = vec![Keypair::from_bytes(&xprog_pair.to_bytes()[..]).unwrap()];
                v.append(&mut xprog_accs.signer_keypairs());
                v
            }
            _other => vec![],
        }
    }
}

/// Trait used on client side to easily validate a program account struct + ix_data for a bare Solana call
pub trait ToInstruction: std::fmt::Debug {
    fn to_ix(
        &self,
        program_id: Pubkey,
        ix_data: &[u8],
    ) -> StdResult<(Instruction, Vec<Keypair>), ErrBox>;

    fn acc_metas(&self, program_id: &Pubkey) -> StdResult<Vec<AccountMeta>, ErrBox>;

    fn signer_keypairs(&self) -> Vec<Keypair>;
}
