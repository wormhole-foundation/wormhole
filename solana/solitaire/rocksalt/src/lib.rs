use proc_macro::TokenStream;
use proc_macro2::TokenStream as TokenStream2;
use quote::quote;
use syn::{
    parse_macro_input,
    parse_quote,
    Data,
    DeriveInput,
    Fields,
    GenericParam,
    Generics,
};

/// Generate a FromAccounts implementation for a product of accounts. Each field is constructed by
/// a call to the Verify::verify instance of its type.
#[proc_macro_derive(FromAccounts)]
pub fn derive_from_accounts(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);
    let name = input.ident;

    // Type params of the instruction context account
    let type_params: Vec<GenericParam> = input
        .generics
        .type_params()
        .map(|v| GenericParam::Type(v.clone()))
        .collect();

    // Generics lifetimes of the peel type
    let mut peel_g = input.generics.clone();
    peel_g.params = parse_quote!('a, 'b: 'a);
    let (_, peel_type_g, _) = peel_g.split_for_impl();

    // Params of the instruction context
    let mut type_generics = input.generics.clone();
    type_generics.params = parse_quote!('b);
    for x in &type_params {
        type_generics.params.push(x.clone());
    }
    let (type_impl_g, type_g, _) = type_generics.split_for_impl();

    // Combined lifetimes of peel and the instruction context
    let mut combined_generics = Generics {
        params: peel_g.params.clone(),
        ..Default::default()
    };
    for x in &type_params {
        combined_generics.params.push(x.clone());
    }
    let (combined_impl_g, _, _) = combined_generics.split_for_impl();

    let from_method = generate_fields(&name, &input.data);
    let persist_method = generate_persist(&input.data);
    let expanded = quote! {
        /// Macro generated implementation of FromAccounts by Solitaire.
        impl #combined_impl_g solitaire::FromAccounts #peel_type_g for #name #type_g {
            fn from<DataType>(pid: &'a solana_program::pubkey::Pubkey, iter: &mut std::slice::Iter<'a, solana_program::account_info::AccountInfo<'b>>, data: &'a DataType) -> solitaire::Result<Self> {
                #from_method
            }
        }

        /// Macro generated implementation of Persist by Solitaire.
        impl #type_impl_g solitaire::Persist for #name #type_g {
            fn persist(&self, program_id: &solana_program::pubkey::Pubkey) -> solitaire::Result<()> {
                #persist_method
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
                            trace!(stringify!(#name));
                            let #name: #ty = solitaire::Peel::peel(&mut solitaire::Context::new(
                                pid,
                                next_account_info(iter)?,
                                data,
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
                        use solitaire::trace;
                        trace!("Peeling:");
                        #(#recurse;)*
                        Ok(#name { #(#names,)* })
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
fn generate_persist(data: &Data) -> TokenStream2 {
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

                        quote! {
                            trace!(stringify!(#name));
                            Peel::persist(&self.#name, program_id)?;
                        }
                    });

                    // Write out our iterator and return the filled structure.
                    quote! {
                        use solitaire::trace;
                        trace!("Persisting:");
                        #(#recurse;)*
                        Ok(())
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
