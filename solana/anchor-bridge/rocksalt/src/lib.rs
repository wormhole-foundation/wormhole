#![allow(warnings)]

mod to_accounts;

use to_accounts::*;

use solana_program::{
    account_info::AccountInfo,
    entrypoint,
    entrypoint::ProgramResult,
    pubkey::Pubkey,
};

use proc_macro::TokenStream;
use proc_macro2::TokenStream as TokenStream2;
use quote::{
    quote,
    quote_spanned,
};
use std::borrow::BorrowMut;
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

#[proc_macro_derive(ToAccounts)]
pub fn derive_to_accounts(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);
    let name = input.ident;
    let to_method_body = generate_to_method(&name, &input.data);

    let expanded = quote! {
    /// Macro-generated implementation of ToAccounts by Solitaire.
    impl<'a> solitaire::ToAccounts for #name<'a> {
        fn to(&self) -> Vec<solana_program::instruction::AccountMeta> {
        #to_method_body
        }
    }
    };

    TokenStream::from(expanded)
}

/// Generate a FromAccounts implementation for a product of accounts. Each field is constructed by
/// a call to the Verify::verify instance of its type.
#[proc_macro_derive(FromAccounts)]
pub fn derive_from_accounts(input: TokenStream) -> TokenStream {
    let mut input = parse_macro_input!(input as DeriveInput);
    let name = input.ident;

    // Type params of the instruction context account
    let type_params: Vec<GenericParam> = input
        .generics
        .type_params()
        .map(|v| GenericParam::Type(v.clone()))
        .collect();

    // Generics lifetimes of the peel type
    let mut peel_g = input.generics.clone();
    peel_g.params = parse_quote!('a, 'b: 'a, 'c);
    let (_, peel_type_g, _) = peel_g.split_for_impl();

    // Params of the instruction context
    let mut type_generics = input.generics.clone();
    type_generics.params = parse_quote!('b);
    for x in &type_params {
        type_generics.params.push(x.clone());
    }
    let (type_impl_g, type_g, _) = type_generics.split_for_impl();

    // Combined lifetimes of peel and the instruction context
    let mut combined_generics = Generics::default();
    combined_generics.params = peel_g.params.clone();
    for x in &type_params {
        combined_generics.params.push(x.clone());
    }
    let (combined_impl_g, _, _) = combined_generics.split_for_impl();

    let from_method = generate_fields(&name, &input.data);
    let persist_method = generate_persist(&name, &input.data);
    let expanded = quote! {
        /// Macro generated implementation of FromAccounts by Solitaire.
        impl #combined_impl_g solitaire::FromAccounts #peel_type_g for #name #type_g {
            fn from<DataType>(pid: &'a solana_program::pubkey::Pubkey, iter: &'c mut std::slice::Iter<'a, solana_program::account_info::AccountInfo<'b>>, data: &'a DataType) -> solitaire::Result<(Self, Vec<solana_program::pubkey::Pubkey>)> {
                #from_method
            }
        }

        impl #combined_impl_g solitaire::Peel<'a, 'b, 'c> for #name #type_g {
            fn peel<I>(ctx: &'c mut Context<'a, 'b, 'c, I>) -> solitaire::Result<Self> where Self: Sized {
                let v: #name #type_g = FromAccounts::from(ctx.this, ctx.iter, ctx.data).map(|v| v.0)?;

                // Verify the instruction constraints
                solitaire::InstructionContext::verify(&v, ctx.this)?;
                // Append instruction level dependencies
                ctx.deps.append(&mut solitaire::InstructionContext::deps(&v));

                Ok(v)
            }
        }

        /// Macro generated implementation of Persist by Solitaire.
        impl #type_impl_g solitaire::Persist for #name #type_g {
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
