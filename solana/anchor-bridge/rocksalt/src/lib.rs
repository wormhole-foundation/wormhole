#![allow(warnings)]

use solana_program::{
    account_info::AccountInfo,
    entrypoint,
    entrypoint::ProgramResult,
    pubkey::Pubkey,
};

use proc_macro::TokenStream;
use proc_macro2::TokenStream as TokenStream2;
use quote::{quote, quote_spanned};
use syn::{
    parse_macro_input,
    parse_quote,
    spanned::Spanned,
    Data,
    DeriveInput,
    Fields,
    GenericParam,
    Generics,
    Index,
};

/// Generate a FromAccounts implementation for a product of accounts. Each field is constructed by
/// a call to the Verify::verify instance of its type.
#[proc_macro_derive(FromAccounts)]
pub fn derive_from_accounts(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);
    let name = input.ident;
    let from_method = generate_fields(&name, &input.data);
    let persist_method = generate_persist(&name, &input.data);
    let expanded = quote! {
        /// Macro generated implementation of FromAccounts by Solitaire.
        impl<'a, 'b: 'a, 'c> solitaire::FromAccounts<'a, 'b, 'c> for #name<'b> {
            fn from<T>(pid: &'a solana_program::pubkey::Pubkey, iter: &'c mut std::slice::Iter<'a, AccountInfo<'b>>, data: &'a T) -> solitaire::Result<(Self, Vec<solana_program::pubkey::Pubkey>)> {
                #from_method
            }
        }

        impl<'a, 'b: 'a, 'c> Peel<'a, 'b, 'c> for #name<'b> {
            fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> solitaire::Result<Self> where Self: Sized {
                let v: #name = FromAccounts::from(ctx.this, ctx.iter, ctx.data).map(|v| v.0)?;

                // Verify the instruction constraints
                solitaire::InstructionContext::verify(&v)?;
                // Append instruction level dependencies
                ctx.deps.append(&mut solitaire::InstructionContext::deps(&v));

                Ok(v)
            }
        }

        /// Macro generated implementation of Persist by Solitaire.
        impl<'a> solitaire::Persist for #name<'a> {
            fn persist(self) {
                use borsh::BorshSerialize;
                //self.guardian_set.serialize(
                //    &mut *self.guardian_set.0.data.borrow_mut()
                //);
            }
        }
    };

    // Hand the output tokens back to the compiler
    TokenStream::from(expanded)
}

/// This function does the heavy lifting of generating the field parsers.
fn generate_fields(name: &syn::Ident, data: &Data) -> TokenStream2 {
    match *data {
        // We only care about structures.
        Data::Struct(ref data) => {
            // We want to inspect its fields.
            match data.fields {
                // For now, we only care about struct { a: T } forms, not struct(T);
                Fields::Named(ref fields) => {
                    // For each field, generate an expression that parses an account info field
                    // from the Solana accounts list. This relies on Verify::verify to do most of
                    // the work.
                    let recurse = fields.named.iter().map(|f| {
                        // Field name, to assign to.
                        let name = &f.ident;
                        let ty = &f.ty;

                        quote! {
                            let #name: #ty = solitaire::Peel::peel(&mut solitaire::Context::new(
                                pid,
                                iter,
                                data,
                                &mut deps,
                            ))?;
                        }
                    });

                    let names = fields.named.iter().map(|f| {
                        let name = &f.ident;
                        quote!(#name)
                    });

                    // Write out our iterator and return the filled structure.
                    quote! {
                        use solana_program::account_info::next_account_info;
                        let mut deps = Vec::new();
                        #(#recurse;)*
                        Ok((#name { #(#names,)* }, deps))
                    }
                }

                Fields::Unnamed(_) => {
                    unimplemented!()
                }

                Fields::Unit => {
                    unimplemented!()
                }
            }
        }

        Data::Enum(_) | Data::Union(_) => unimplemented!(),
    }
}

/// This function does the heavy lifting of generating the field parsers.
fn generate_persist(name: &syn::Ident, data: &Data) -> TokenStream2 {
    match *data {
        // We only care about structures.
        Data::Struct(ref data) => {
            // We want to inspect its fields.
            match data.fields {
                // For now, we only care about struct { a: T } forms, not struct(T);
                Fields::Named(ref fields) => {
                    // For each field, generate an expression that parses an account info field
                    // from the Solana accounts list. This relies on Verify::verify to do most of
                    // the work.
                    let recurse = fields.named.iter().map(|f| {
                        // Field name, to assign to.
                        let name = &f.ident;
                        let ty = &f.ty;

                        quote! {
                            let #name: #ty = Peel::peel(&mut solitaire::Context::new(
                                pid,
                                iter,
                                data,
                                &mut deps,
                            ))?;
                        }
                    });

                    let names = fields.named.iter().map(|f| {
                        let name = &f.ident;
                        quote!(#name)
                    });

                    // Write out our iterator and return the filled structure.
                    quote! {
                        use solana_program::account_info::next_account_info;
                        let mut deps = Vec::new();
                        #(#recurse;)*
                        Ok((#name { #(#names,)* }, deps))
                    }
                }

                Fields::Unnamed(_) => {
                    unimplemented!()
                }

                Fields::Unit => {
                    unimplemented!()
                }
            }
        }

        Data::Enum(_) | Data::Union(_) => unimplemented!(),
    }
}
