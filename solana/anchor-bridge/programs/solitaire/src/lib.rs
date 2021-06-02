#![feature(const_generics)]
#![feature(const_generics_defaults)]
#![allow(warnings)]

pub mod seeded;

pub use seeded::*;
pub use rocksalt::*;

// Lacking:
//
// - Error is a lacking as its just a basic enum, maybe use errorcode.
// - Client generation incomplete.

// We need a few Solana things in scope in order to properly abstract Solana.
pub use solana_program::{
    account_info::AccountInfo,
    entrypoint,
    entrypoint::ProgramResult,
    pubkey::Pubkey,
    system_instruction,
    system_program,
    sysvar::{self, SysvarId},
};

// Later on we will define types that don't actually contain data, PhantomData will help us.
pub use std::marker::PhantomData;

// We'll need these traits to make any wrappers we define more ergonomic for users.
pub use std::ops::{Deref, DerefMut};

// Borsh is Solana's goto serialization target, so we'll need this if we want to do any
// serialization on the users behalf.
pub use borsh::{BorshDeserialize, BorshSerialize};
use solana_program::{
    account_info::next_account_info,
    instruction::AccountMeta,
    program::invoke_signed,
    program_error::ProgramError,
    program_pack::Pack,
    rent::Rent,
};
use std::{
    io::{ErrorKind, Write},
    slice::Iter,
    string::FromUtf8Error,
};

/// There are several places in Solitaire that might fail, we want descriptive errors.
#[derive(Debug)]
pub enum SolitaireError {
    /// The AccountInfo parser expected a Signer, but the account did not sign.
    InvalidSigner(Pubkey),

    /// The AccountInfo parser expected a Sysvar, but the key was invalid.
    InvalidSysvar(Pubkey),

    /// The AccountInfo parser tried to derive the provided key, but it did not match.
    InvalidDerive(Pubkey),

    /// The instruction payload itself could not be deserialized.
    InstructionDeserializeFailed,

    /// An IO error was captured, wrap it up and forward it along.
    IoError(std::io::Error),

    /// An solana program error
    ProgramError(ProgramError),

    Custom(u64),
}

impl From<ProgramError> for SolitaireError {
    fn from(e: ProgramError) -> Self {
        SolitaireError::ProgramError(e)
    }
}

impl From<std::io::Error> for SolitaireError {
    fn from(e: std::io::Error) -> Self {
        SolitaireError::IoError(e)
    }
}

impl Into<ProgramError> for SolitaireError {
    fn into(self) -> ProgramError {
        if let SolitaireError::ProgramError(e) = self {
            return e;
        }
        // TODO
        ProgramError::Custom(0)
    }
}

/// Quality of life Result type for the Solitaire stack.
pub type Result<T> = std::result::Result<T, SolitaireError>;
pub type ErrBox = Box<dyn std::error::Error>;

pub trait Persist {
    fn persist(self);
}

#[repr(transparent)]
pub struct Packed<T: Pack + solana_program::program_pack::IsInitialized>(T);

impl<T: Pack + solana_program::program_pack::IsInitialized> BorshDeserialize for Packed<T> {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        Ok(Packed(
            T::unpack(buf).map_err(|e| std::io::Error::new(ErrorKind::Other, e))?,
        ))
    }
}

impl<T: Pack + solana_program::program_pack::IsInitialized> BorshSerialize for Packed<T> {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        let mut data: [u8; 2000] = [0u8; 2000];
        T::pack_into_slice(self, &mut data);
        writer.write(&data);

        Ok(())
    }
}

impl<T: Pack + solana_program::program_pack::IsInitialized> Deref for Packed<T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

// The following set of recursive types form the basis of this library, each one represents a
// constraint to be fulfilled during parsing an account info.

#[repr(transparent)]
pub struct Signer<Next>(Next);

#[repr(transparent)]
pub struct System<Next>(Next);

#[repr(transparent)]
pub struct Sysvar<Next, Var>(Next, PhantomData<Var>);

#[repr(transparent)]
pub struct Derive<Next, const Seed: &'static str>(Next);

/// A shorter type alias for AccountInfo to make types slightly more readable.
pub type Info<'r> = AccountInfo<'r>;

/// Another alias for an AccountInfo that pairs it with the deserialized data that resides in the
/// accounts storage.
///
/// Note on const generics:
///
/// Solana's Rust version is JUST old enough that it cannot use constant variables in its default
/// parameter assignments. But these DO work in the consumption side so a user can still happily
/// use this type by writing for example:
///
/// Data<(), Uninitialized, Lazy>
///
/// But here, we must write `Lazy: bool = true` for now unfortunately.
#[rustfmt::skip]
pub struct Data < 'r, T, const IsInitialized: bool = true, const Lazy: bool = false > (
pub Info<'r >,
pub T,
);

/// A tag for accounts that should be deserialized lazily.
pub const Lazy: bool = true;

/// A tag for accounts that should be deserialized immediately (the default).
pub const Strict: bool = false;

/// A tag for accounts that are expected to have already been initialized.
pub const Initialized: bool = true;

/// A tag for accounts that must be uninitialized.
pub const Uninitialized: bool = false;

/// The context is threaded through each check. Include anything within this structure that you
/// would like to have access to as each layer of dependency is peeled off.
pub struct Context<'a, 'b: 'a, 'c, T> {
    /// A reference to the program_id of the current program.
    pub this: &'a Pubkey,

    pub iter: &'c mut Iter<'a, AccountInfo<'b>>,

    /// This is a reference to the instruction data we are processing this
    /// account for.
    pub data: &'a T,

    /// This is a list of dependent keys that are emitted by this verification pipeline. This
    /// allows things such as `rent`/`system` to be emitted as required for an account without
    /// having to specify them in the original instruction account data.
    pub deps: &'c mut Vec<Pubkey>,

    info: Option<&'a AccountInfo<'b>>,
}

pub struct ExecutionContext<'a, 'b: 'a> {
    /// A reference to the program_id of the current program.
    pub program_id: &'a Pubkey,

    /// All accounts passed into the program
    pub accounts: &'a [AccountInfo<'b>],
}

impl<'a, 'b: 'a, 'c, T> Context<'a, 'b, 'c, T> {
    pub fn new(
        program: &'a Pubkey,
        iter: &'c mut Iter<'a, AccountInfo<'b>>,
        data: &'a T,
        deps: &'c mut Vec<Pubkey>,
    ) -> Self {
        Context {
            this: program,
            iter,
            data,
            deps,
            info: None,
        }
    }

    pub fn info<'d>(&'d mut self) -> &'a AccountInfo<'b> {
        match self.info {
            None => {
                let info = next_account_info(self.iter).unwrap();
                self.info = Some(info);
                info
            }
            Some(v) => v,
        }
    }
}

impl<'r, T, const IsInitialized: bool, const Lazy: bool> Deref
for Data<'r, T, IsInitialized, Lazy>
{
    type Target = T;
    fn deref(&self) -> &Self::Target {
        &self.1
    }
}

impl<'r, T, const IsInitialized: bool, const Lazy: bool> DerefMut
for Data<'r, T, IsInitialized, Lazy>
{
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.1
    }
}

impl<T> Deref for Signer<T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T> DerefMut for Signer<T> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl<T, Var> Deref for Sysvar<T, Var> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T, Var> DerefMut for Sysvar<T, Var> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl<T> Deref for System<T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T> DerefMut for System<T> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl<T, const Seed: &'static str> Deref for Derive<T, Seed> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl<T, const Seed: &'static str> DerefMut for Derive<T, Seed> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

pub trait Keyed {
    fn pubkey(&self) -> &Pubkey;
}

impl<'r, T, const IsInitialized: bool, const Lazy: bool> Keyed
for Data<'r, T, IsInitialized, Lazy>
{
    fn pubkey(&self) -> &Pubkey {
        self.0.key
    }
}

impl<T> Keyed for Signer<T>
    where
        T: Keyed,
{
    fn pubkey(&self) -> &Pubkey {
        self.0.pubkey()
    }
}

impl<T, Var> Keyed for Sysvar<T, Var>
    where
        T: Keyed,
{
    fn pubkey(&self) -> &Pubkey {
        self.0.pubkey()
    }
}

impl<T> Keyed for System<T>
    where
        T: Keyed,
{
    fn pubkey(&self) -> &Pubkey {
        self.0.pubkey()
    }
}

impl<T, const Seed: &'static str> Keyed for Derive<T, Seed>
    where
        T: Keyed,
{
    fn pubkey(&self) -> &Pubkey {
        self.0.pubkey()
    }
}

impl<'r> Keyed for Info<'r> {
    fn pubkey(&self) -> &Pubkey {
        self.key
    }
}

/// Lamports to pay to an account being created
pub enum CreationLamports {
    Exempt,
    Amount(u64),
}

impl CreationLamports {
    /// Amount of lamports to be paid in account creation
    pub fn amount(self, size: usize) -> u64 {
        match self {
            CreationLamports::Exempt => Rent::default().minimum_balance(size),
            CreationLamports::Amount(v) => v,
        }
    }
}

impl<const Seed: &'static str> Derive<AccountInfo<'_>, Seed> {
    pub fn create(
        &self,
        ctx: &ExecutionContext,
        payer: &Pubkey,
        lamports: CreationLamports,
        space: usize,
        owner: &Pubkey,
    ) -> Result<()> {
        let ix = system_instruction::create_account(
            payer,
            self.0.key,
            lamports.amount(space),
            space as u64,
            owner,
        );
        Ok(invoke_signed(&ix, ctx.accounts, &[&[Seed.as_bytes()]])?)
    }
}

impl<const Seed: &'static str, T: BorshSerialize> Derive<Data<'_, T, Uninitialized>, Seed> {
    pub fn create(
        &self,
        ctx: &ExecutionContext,
        payer: &Pubkey,
        lamports: CreationLamports,
    ) -> Result<()> {
        // Get serialized struct size
        let size = self.0.try_to_vec().unwrap().len();
        let ix = system_instruction::create_account(
            payer,
            self.0.0.key,
            lamports.amount(size),
            size as u64,
            ctx.program_id,
        );
        Ok(invoke_signed(&ix, ctx.accounts, &[&[Seed.as_bytes()]])?)
    }
}

/// Generic Peel trait. This provides a way to describe what each "peeled"
/// layer of our constraints should check.
pub trait Peel<'a, 'b: 'a, 'c> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self>
        where
            Self: Sized;
}

/// Peel a Derived Key
impl<'a, 'b: 'a, 'c, T: Peel<'a, 'b, 'c>, const Seed: &'static str> Peel<'a, 'b, 'c>
for Derive<T, Seed>
{
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        // Attempt to Derive Seed
        let (derived, bump) = Pubkey::find_program_address(&[Seed.as_ref()], ctx.this);
        match derived == *ctx.info().key {
            true => T::peel(ctx).map(|v| Derive(v)),
            _ => Err(SolitaireError::InvalidDerive(*ctx.info().key).into()),
        }
    }
}

/// Peel a Signer.
impl<'a, 'b: 'a, 'c, T: Peel<'a, 'b, 'c>> Peel<'a, 'b, 'c> for Signer<T> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        match ctx.info().is_signer {
            true => T::peel(ctx).map(|v| Signer(v)),
            _ => Err(SolitaireError::InvalidSigner(*ctx.info().key).into()),
        }
    }
}

/// Expicitly depend upon the System account.
impl<'a, 'b: 'a, 'c, T: Peel<'a, 'b, 'c>> Peel<'a, 'b, 'c> for System<T> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        match true {
            true => T::peel(ctx).map(|v| System(v)),
            _ => panic!(),
        }
    }
}

/// Peel a Sysvar
impl<'a, 'b: 'a, 'c, T: Peel<'a, 'b, 'c>, Var: SysvarId> Peel<'a, 'b, 'c> for Sysvar<T, Var> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        match <Var as SysvarId>::check_id(ctx.info().key) {
            true => T::peel(ctx).map(|v| Sysvar(v, PhantomData)),
            _ => Err(SolitaireError::InvalidSysvar(*ctx.info().key).into()),
        }
    }
}

/// This is our structural recursion base case, the trait system will stop
/// generating new nested calls here.
impl<'a, 'b: 'a, 'c> Peel<'a, 'b, 'c> for Info<'b> {
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        Ok(ctx.info().clone())
    }
}

/// This is our structural recursion base case, the trait system will stop
/// generating new nested calls here.
impl<'a, 'b: 'a, 'c, T: BorshDeserialize, const IsInitialized: bool, const Lazy: bool>
Peel<'a, 'b, 'c> for Data<'b, T, IsInitialized, Lazy>
{
    fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> Result<Self> {
        // If we're initializing the type, we should emit system/rent as deps.
        if !IsInitialized {
            ctx.deps.push(sysvar::rent::ID);
            ctx.deps.push(system_program::ID);
        }

        let data = T::try_from_slice(&mut *ctx.info().data.borrow_mut())?;

        Ok(Data(ctx.info().clone(), data))
    }
}

pub trait Wrap {
    fn wrap(&self) -> Vec<AccountMeta>;
}

impl<T> Wrap for T
    where
        T: ToAccounts,
{
    fn wrap(&self) -> Vec<AccountMeta> {
        self.to()
    }
}

impl<T> Wrap for Signer<T> {
    fn wrap(&self) -> Vec<AccountMeta> {
        todo!()
    }
}

impl<T, const Seed: &'static str> Wrap for Derive<T, Seed> {
    fn wrap(&self) -> Vec<AccountMeta> {
        todo!()
    }
}

impl<'a, T: BorshSerialize, const IsInitialized: bool, const Lazy: bool> Wrap
for Data<'a, T, IsInitialized, Lazy>
{
    fn wrap(&self) -> Vec<AccountMeta> {
        todo!()
    }
}

pub trait InstructionContext<'a> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        Ok(())
    }

    fn deps(&self) -> Vec<Pubkey> {
        vec![]
    }
}

/// Trait definition that describes types that can be constructed from a list of solana account
/// references. A list of dependent accounts is produced as a side effect of the parsing stage.
pub trait FromAccounts<'a, 'b: 'a, 'c> {
    fn from<T>(
        _: &'a Pubkey,
        _: &'c mut Iter<'a, AccountInfo<'b>>,
        _: &'a T,
    ) -> Result<(Self, Vec<Pubkey>)>
        where
            Self: Sized;
}

pub trait ToAccounts {
    fn to(&self) -> Vec<AccountMeta>;
}

/// This is our main codegen macro. It takes as input a list of enum-like variants mapping field
/// types to function calls. The generated code produces:
///
/// - An `Instruction` enum with the enum variants passed in.
/// - A set of functions which take as arguments the enum fields.
/// - A Dispatcher that deserializes bytes into the enum and dispatches the function call.
/// - A set of client calls scoped to the module `api` that can generate instructions.
#[macro_export]
macro_rules! solitaire {
    { $($row:ident($kind:ty) => $fn:ident),+ $(,)* } => {
        mod instruction {
            use super::*;
            use borsh::{BorshDeserialize, BorshSerialize};
            use solitaire::{Persist, FromAccounts, Result};

            /// Generated:
            /// This Instruction contains a 1-1 mapping for each enum variant to function call. The
            /// function calls can be found below in the `api` module.

            #[derive(BorshSerialize, BorshDeserialize)]
            enum Instruction {
                $($row($kind),)*
            }

            /// This entrypoint is generated from the enum above, it deserializes incoming bytes
            /// and automatically dispatches to the correct method.
            pub fn dispatch<'a, 'b: 'a, 'c>(p: &Pubkey, a: &'c [AccountInfo<'b>], d: &[u8]) -> Result<()> {
                match BorshDeserialize::try_from_slice(d).map_err(|_| SolitaireError::InstructionDeserializeFailed)? {
                    $(
                        Instruction::$row(ix_data) => {
                            let (mut accounts, _deps): ($row, _) = FromAccounts::from(p, &mut a.iter(), &()).unwrap();
                            $fn(&ExecutionContext{program_id: p, accounts: a}, &mut accounts, ix_data)?;
                            accounts.persist();
                            Ok(())
                        }
                    )*

                    _ => {
                        Ok(())
                    }
                }
            }

            pub fn solitaire<'a, 'b: 'a>(p: &Pubkey, a: &'a [AccountInfo<'b>], d: &[u8]) -> ProgramResult {
                if let Err(err) = dispatch(p, a, d) {
                }
                Ok(())
            }
        }

        /// This module contains a 1-1 mapping for each function to an enum variant. The variants
        /// can be matched to the Instruction found above.
        mod client {
            use super::*;
            use solana_program::instruction::Instruction;

            /// Generated from Instruction Field
            $(pub(crate) fn $fn(pid: &Pubkey, accounts: $row, ix_data: $kind) -> std::result::Result<Instruction, ErrBox> {
                Ok(Instruction {
                    program_id: *pid,
                    accounts: vec![],
                    data: ix_data.try_to_vec()?,
                })
            })*
        }

        use instruction::solitaire;
        entrypoint!(solitaire);
    }
}

#[macro_export]
macro_rules! info_wrapper {
    ($name:ident) => {
        pub struct $name<'b>(Info<'b>);

        impl<'b> Deref for $name<'b> {
            type Target = Info<'b>;

            fn deref(&self) -> &Self::Target {
                return &self.0;
            }
        }

        impl<'b> Keyed for $name<'b> {
            fn pubkey(&self) -> &Pubkey {
                self.key
            }
        }

        impl<'a, 'b: 'a, 'c> Peel<'a, 'b, 'c> for $name<'b> {
            fn peel<T>(ctx: &'c mut Context<'a, 'b, 'c, T>) -> Result<Self>
            where
                Self: Sized,
            {
                return Ok($name(ctx.info().clone()));
            }
        }
    };
    ($name:ident, size: $size:expr) => {
        #[repr(transparent)]
        pub struct $name<'b>(Info<'b>);

        impl<'b> Deref for $name<'b> {
            type Target = Info<'b>;

            fn deref(&self) -> &Self::Target {
                unsafe { std::mem::transmute(&self.0) }
            }
        }

        impl<'b> DerefMut for $name<'b> {
            fn deref_mut(&mut self) -> &mut Self::Target {
                unsafe { std::mem::transmute(&mut self.0) }
            }
        }

        impl<'b> AccountSize for $name<'b> {
            fn size(&self) -> usize {
                return $size;
            }
        }

        impl<'b> Keyed for $name<'b> {
            fn pubkey(&self) -> &Pubkey {
                self.key
            }
        }

        impl<'a, 'b: 'a, 'c> Peel<'a, 'b, 'c> for $name<'b> {
            fn peel<T>(ctx: &'c mut Context<'a, 'b, 'c, T>) -> Result<Self>
            where
                Self: Sized,
            {
                Ok($name(ctx.info().clone()))
            }
        }
    };
}

#[macro_export]
macro_rules! data_wrapper {
    ($name:ident, $embed:ty) => {
        #[repr(transparent)]
        pub struct $name<'b>(Data<'b, $embed>);

        impl<'b> Deref for $name<'b> {
            type Target = Data<'b, $embed>;

            fn deref(&self) -> &Self::Target {
                return &self.0;
            }
        }

        impl<'b> DerefMut for $name<'b> {
            fn deref_mut(&mut self) -> &mut Self::Target {
                unsafe { std::mem::transmute(&mut self.0) }
            }
        }

        impl<'b> Keyed for $name<'b> {
            fn pubkey(&self) -> &Pubkey {
                self.0.pubkey()
            }
        }

        impl<'b> AccountSize for $name<'b> {
            fn size(&self) -> usize {
                return self.0.size();
            }
        }

        impl<'a, 'b: 'a, 'c> Peel<'a, 'b, 'c> for $name<'b> {
            fn peel<T>(ctx: &'c mut Context<'a, 'b, 'c, T>) -> Result<Self>
            where
                Self: Sized,
            {
                Data::peel(ctx).map(|v| $name(v))
            }
        }
    };
}
