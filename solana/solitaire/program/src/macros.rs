use std::ops::{
    Deref,
    DerefMut,
};

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
    { $($row:ident($kind:ty) => $fn:ident),+ $(,)* } => {
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
                        n if n == Instruction::$row as u8 => {
                            (move || {
                                trace!("Dispatch: {}", stringify!($row));
                                let ix_data: $kind = BorshDeserialize::try_from_slice(&d[1..]).map_err(|e| SolitaireError::InstructionDeserializeFailed(e))?;
                                let mut accounts: $row = FromAccounts::from(p, &mut a.iter(), &())?;
                                $fn(&ExecutionContext{program_id: p, accounts: a}, &mut accounts, ix_data)?;
                                Persist::persist(&accounts, p)?;
                                Ok(())
                            })()
                        },
                    )*

                    other => {
                        Err(SolitaireError::UnknownInstruction(other))
                    }
                }
            }

            pub fn solitaire<'a, 'b: 'a>(p: &Pubkey, a: &'a [AccountInfo<'b>], d: &[u8]) -> ProgramResult {
                trace!("{} {} built with {}", env!("CARGO_PKG_NAME"), env!("CARGO_PKG_VERSION"), solitaire::PKG_NAME_VERSION);
                if let Err(err) = dispatch(p, a, d) {
                    trace!("Error: {:?}", err);
                    return Err(err.into());
                }
                Ok(())
            }
        }

        use instruction::solitaire;
        #[cfg(not(feature = "no-entrypoint"))]
        solana_program::entrypoint!(solitaire);
    }
}

#[macro_export]
macro_rules! data_wrapper {
    ($name:ident, $embed:ty, $state:expr) => {
        #[repr(transparent)]
        pub struct $name<'b>(solitaire::Data<'b, $embed, { $state }>);

        impl<'b> std::ops::Deref for $name<'b> {
            type Target = solitaire::Data<'b, $embed, { $state }>;

            fn deref(&self) -> &Self::Target {
                return &self.0;
            }
        }

        impl<'b> std::ops::DerefMut for $name<'b> {
            fn deref_mut(&mut self) -> &mut Self::Target {
                unsafe { std::mem::transmute(&mut self.0) }
            }
        }

        impl<'a, 'b: 'a> solitaire::processors::keyed::Keyed<'a, 'b> for $name<'b> {
            fn info(&'a self) -> &'a solitaire::Info<'b> {
                self.0.info()
            }
        }

        impl<'b> solitaire::AccountSize for $name<'b> {
            fn size(&self) -> usize {
                return self.0.size();
            }
        }

        impl<'a, 'b: 'a, 'c> solitaire::Peel<'a, 'b, 'c> for $name<'b> {
            fn peel<T>(ctx: &'c mut solitaire::Context<'a, 'b, 'c, T>) -> solitaire::Result<Self>
            where
                Self: Sized,
            {
                solitaire::Data::peel(ctx).map(|v| $name(v))
            }

            fn deps() -> Vec<solana_program::pubkey::Pubkey> {
                solitaire::Data::<'_, $embed, { $state }>::deps()
            }

            fn persist(
                &self,
                program_id: &solana_program::pubkey::Pubkey,
            ) -> solitaire::Result<()> {
                solitaire::Data::<'_, $embed, { $state }>::persist(self, program_id)
            }
        }

        impl<'b> solitaire::Owned for $name<'b> {
            fn owner(&self) -> solitaire::AccountOwner {
                return self.1.owner();
            }
        }

        #[cfg(feature = "client")]
        impl<'b> solitaire_client::Wrap for $name<'b> {
            fn wrap(
                a: &solitaire_client::AccEntry,
            ) -> std::result::Result<Vec<solitaire_client::AccountMeta>, solitaire_client::ErrBox>
            {
                solitaire::Data::<'b, $embed, { $state }>::wrap(a)
            }
        }
    };
}

#[macro_export]
macro_rules! pack_type {
    ($name:ident, $embed:ty, $owner:expr) => {
        #[repr(transparent)]
        pub struct $name(pub $embed);

        impl BorshDeserialize for $name {
            fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
                let acc = $name(
                    solana_program::program_pack::Pack::unpack(buf)
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
                writer.write(&data);

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
