extern crate proc_macro;

use quote::quote;
use syn::parse_macro_input;

/// Borrowed from anchor-attribute-account (crate version 0.27.0).
///
/// An attribute for a data structure representing a Solana account.
///
/// `#[legacy_account]` generates trait implementations for the following traits:
///
/// - [`AccountSerialize`](./trait.AccountSerialize.html)
/// - [`AccountDeserialize`](./trait.AccountDeserialize.html)
/// - [`AnchorSerialize`](./trait.AnchorSerialize.html)
/// - [`AnchorDeserialize`](./trait.AnchorDeserialize.html)
/// - [`Clone`](https://doc.rust-lang.org/std/clone/trait.Clone.html)
/// - [`Owner`](./trait.Owner.html)
///
/// Unlike Anchor's `#[account]` macro, these accounts serialize with a legacy discriminator.
#[proc_macro_attribute]
pub fn legacy_account(
    args: proc_macro::TokenStream,
    input: proc_macro::TokenStream,
) -> proc_macro::TokenStream {
    let args_str = args.to_string();
    if args_str.split(',').count() > 1 {
        panic!("Legacy account attribute takes no arguments.")
    }

    let account_strct = parse_macro_input!(input as syn::ItemStruct);
    let account_name = &account_strct.ident;
    let account_name_str = account_name.to_string();
    let (impl_gen, type_gen, where_clause) = account_strct.generics.split_for_impl();

    proc_macro::TokenStream::from({
        quote! {
            #[derive(AnchorSerialize, AnchorDeserialize, Clone)]
            #account_strct

            #[automatically_derived]
            impl #impl_gen anchor_lang::AccountSerialize for #account_name #type_gen #where_clause {
                fn try_serialize<W: std::io::Write>(&self, writer: &mut W) -> anchor_lang::Result<()> {
                    if writer.write_all(&Self::LEGACY_DISCRIMINATOR).is_err() {
                        return Err(anchor_lang::error::ErrorCode::AccountDidNotSerialize.into());
                    }

                    if AnchorSerialize::serialize(self, writer).is_err() {
                        return Err(anchor_lang::error::ErrorCode::AccountDidNotSerialize.into());
                    }
                    Ok(())
                }
            }

            #[automatically_derived]
            impl #impl_gen anchor_lang::AccountDeserialize for #account_name #type_gen #where_clause {
                fn try_deserialize(buf: &mut &[u8]) -> anchor_lang::Result<Self> {
                    let disc_len = Self::LEGACY_DISCRIMINATOR.len();
                    if buf.len() < disc_len {
                        return Err(anchor_lang::error::ErrorCode::AccountDiscriminatorNotFound.into());
                    };
                    let given_disc = &buf[..disc_len];
                    if Self::LEGACY_DISCRIMINATOR != *given_disc {
                        return Err(anchor_lang::error!(
                            anchor_lang::error::ErrorCode::AccountDiscriminatorMismatch
                        )
                        .with_account_name(#account_name_str));
                    }
                    Self::try_deserialize_unchecked(buf)
                }

                fn try_deserialize_unchecked(buf: &mut &[u8]) -> anchor_lang::Result<Self> {
                    let mut data: &[u8] = &buf[Self::LEGACY_DISCRIMINATOR.len()..];
                    AnchorDeserialize::deserialize(&mut data)
                        .map_err(|_| anchor_lang::error::ErrorCode::AccountDidNotDeserialize.into())
                }
            }

            #[automatically_derived]
            impl #impl_gen anchor_lang::Owner for #account_name #type_gen #where_clause {
                fn owner() -> Pubkey {
                    crate::ID
                }
            }
        }
    })
}
