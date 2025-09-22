/// A wrapper around Solana's `msg!` macro that is a no-op by default, allows for adding traces
/// through the application that can be toggled during tests.
#[macro_export]
macro_rules! trace {
    ( $($arg:tt)* ) => { $crate::trace_impl!( $($arg)* ) };
}

#[cfg(feature = "trace")]
#[macro_export]
macro_rules! trace_impl {
    ( $($arg:tt)* ) => { solana_program::msg!( $($arg)* ) };
}

#[cfg(not(feature = "trace"))]
#[macro_export]
macro_rules! trace_impl {
    ( $($arg:tt)* ) => {};
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
    { $($row:ident => $fn:ident),+ $(,)* } => {
        pub mod instruction {
            use super::*;
            use borsh::{
                BorshDeserialize,
                BorshSerialize,
            };
            use solana_program::{
                account_info::AccountInfo,
                entrypoint::ProgramResult,
                program_error::ProgramError,
                pubkey::Pubkey,
            };
            use solitaire::{
                trace,
                ExecutionContext,
                FromAccounts,
                Persist,
                Result,
                SolitaireError,
            };

            $(
                // Generated module wrapping instruction handler.
                //
                // These are needed to force the compiler to generate a new function that has not
                // been inlined, this provides a new stack frame. Without this, the stack frame for
                // deserialization and the handler is the same as that used by solitaire, leading
                // to bust stacks.
                #[allow(non_snake_case)]
                pub mod $row {
                    use super::*;

                    #[inline(never)]
                    pub fn execute<'a, 'b: 'a, 'c>(p: &Pubkey, a: &'c [AccountInfo<'b>], d: &[u8]) -> Result<()> {
                        let ix_data = BorshDeserialize::try_from_slice(d).map_err(|e| SolitaireError::InstructionDeserializeFailed(e))?;
                        let mut accounts = FromAccounts::from(p, &mut a.iter(), &())?;
                        $fn(&ExecutionContext{program_id: p, accounts: a}, &mut accounts, ix_data)?;
                        Persist::persist(accounts.as_ref(), p)?;
                        Ok(())
                    }
                }
            )*

            /// Generated:
            /// This Instruction contains a 1-1 mapping for each enum variant to function call. The
            /// function calls can be found below in the `api` module.
            #[repr(u8)]
            #[derive(BorshSerialize, BorshDeserialize)]
            pub enum Instruction {
                $($row,)*
            }

            /// This entrypoint is generated from the enum above, it deserializes incoming bytes
            /// and automatically dispatches to the correct method.
            pub fn dispatch<'a, 'b: 'a, 'c>(p: &Pubkey, a: &'c [AccountInfo<'b>], d: &[u8]) -> Result<()> {
                match d[0] {
                    $(
                        n if n == Instruction::$row as u8 => $row::execute(p, a, &d[1..]),
                    )*

                    other => {
                        Err(SolitaireError::UnknownInstruction(other))
                    }
                }
            }

            pub fn solitaire(p: &Pubkey, a: &[AccountInfo], d: &[u8]) -> ProgramResult {
                trace!("{} {} built with {}", env!("CARGO_PKG_NAME"), env!("CARGO_PKG_VERSION"), solitaire::PKG_NAME_VERSION);
                if let Err(err) = dispatch(p, a, d) {
                    solana_program::msg!("Error: {:?}", err);
                    return Err(err.into());
                }
                Ok(())
            }
        }

        pub use instruction::solitaire;
        #[cfg(not(feature = "no-entrypoint"))]
        solana_program::entrypoint!(solitaire);
    }
}

#[macro_export]
macro_rules! pack_type_impl {
    // We take a "unpacker" as an input, which specifies how to unpack the embedded type.
    // In most cases, this should be just be
    // `solana_program::program_pack::Pack`, but in some cases (like token-2022
    // mints) it may be a custom trait that provides an `unpack` method. This is
    // because `Pack` does a strict length check on the account, whereas
    // token-2022 mints with extensions might be longer.
    //
    // NOTE: we only use this on the deserialisation side, but we keep the call for serialisation
    // as solana_program::program_pack::Pack::pack_into_slice. We could generalise that side too, but in
    // reality, that code is never invoked, because solitaire will persist (and
    // thus serialise) accounts that are owned by the current program.
    // `pack_type!` on the other hands is only used for solitaire-ising external accounts.
    ($name:ident, $embed:ty, $owner:expr, $unpacker:path) => {
        #[repr(transparent)]
        pub struct $name(pub $embed);

        impl BorshDeserialize for $name {
            fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
                let acc = $name(
                    <$embed as $unpacker>::unpack(buf)
                        .map_err(|e| std::io::Error::new(std::io::ErrorKind::Other, e))?,
                );
                // We need to clear the buf to show to Borsh that we've read all data
                *buf = &buf[..0];

                Ok(acc)
            }
        }

        impl BorshSerialize for $name {
            fn serialize<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
                let mut data = [0u8; <$embed as solana_program::program_pack::Pack>::LEN];
                solana_program::program_pack::Pack::pack_into_slice(&self.0, &mut data);
                writer.write_all(&data)?;

                Ok(())
            }
        }

        impl solitaire::processors::seeded::Owned for $name {
            fn owner(&self) -> solitaire::processors::seeded::AccountOwner {
                return $owner;
            }
        }

        impl std::ops::Deref for $name {
            type Target = $embed;
            fn deref(&self) -> &Self::Target {
                unsafe { std::mem::transmute(&self.0) }
            }
        }

        impl std::default::Default for $name {
            fn default() -> Self {
                $name(<$embed>::default())
            }
        }
    };
}

#[macro_export]
macro_rules! pack_type {
    ($name:ident, $embed:ty, AccountOwner::OneOf($owner:expr)) => {
        solitaire::pack_type_impl!(
            $name,
            $embed,
            AccountOwner::OneOf($owner),
            solana_program::program_pack::Pack
        );

        impl solitaire::processors::seeded::MultiOwned for $name {
        }
    };
    ($name:ident, $embed:ty, $owner:expr) => {
        solitaire::pack_type_impl!($name, $embed, $owner, solana_program::program_pack::Pack);

        impl solitaire::processors::seeded::SingleOwned for $name {
        }
    };
    ($name:ident, $embed:ty, AccountOwner::OneOf($owner:expr), $unpacker:ident) => {
        solitaire::pack_type_impl!($name, $embed, AccountOwner::OneOf($owner), $unpacker);

        impl solitaire::processors::seeded::MultiOwned for $name {
        }
    };
}
