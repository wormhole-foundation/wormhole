use std::ops::{
    Deref,
    DerefMut,
};

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
            use borsh::{BorshDeserialize, BorshSerialize};
            use solana_program::{
                account_info::AccountInfo,
                entrypoint::ProgramResult,
		program_error::ProgramError,
                pubkey::Pubkey,
            };
            use solitaire::{FromAccounts, Persist, Result};

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
		solana_program::msg!(concat!(env!("CARGO_PKG_NAME"), " ", env!("CARGO_PKG_VERSION")));
                if let Err(err) = dispatch(p, a, d) {

		    solana_program::msg!("Error: {:?}", err);
		    return Err(err.into());
                }
                Ok(())
            }
        }

        /// This module contains a 1-1 mapping for each function to an enum variant. The variants
        /// can be matched to the Instruction found above.
        pub mod client {
            use super::*;
            use borsh::BorshSerialize;
            use solana_program::{instruction::Instruction, pubkey::Pubkey};

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
        solana_program::entrypoint!(solitaire);
    }
}

#[macro_export]
macro_rules! data_wrapper {
    ($name:ident, $embed:ty, $state:expr) => {
        #[repr(transparent)]
        pub struct $name<'b>(Data<'b, $embed, { $state }>);

        impl<'b> std::ops::Deref for $name<'b> {
            type Target = Data<'b, $embed, { $state }>;

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
            fn info(&'a self) -> &'a Info<'b> {
                self.0.info()
            }
        }

        impl<'b> solitaire::processors::seeded::AccountSize for $name<'b> {
            fn size(&self) -> usize {
                return self.0.size();
            }
        }

        impl<'a, 'b: 'a, 'c> solitaire::Peel<'a, 'b, 'c> for $name<'b> {
            fn peel<T>(ctx: &'c mut Context<'a, 'b, 'c, T>) -> Result<Self>
            where
                Self: Sized,
            {
                Data::peel(ctx).map(|v| $name(v))
            }
        }

        impl<'b> solitaire::processors::seeded::Owned for $name<'b> {
            fn owner(&self) -> solitaire::processors::seeded::AccountOwner {
                return self.1.owner();
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
                Ok($name(
                    solana_program::program_pack::Pack::unpack(buf)
                        .map_err(|e| std::io::Error::new(std::io::ErrorKind::Other, e))?,
                ))
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
