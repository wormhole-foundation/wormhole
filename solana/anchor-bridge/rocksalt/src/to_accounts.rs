//! Derive macro logic for ToAccounts

use proc_macro::TokenStream;
use proc_macro2::TokenStream as TokenStream2;
use quote::{quote, quote_spanned};
use syn::{
    parse_macro_input,
    parse_quote,
    spanned::Spanned,
    Data,
    DataStruct,
    DeriveInput,
    Fields,
    GenericParam,
    Generics,
    Index,
};

pub fn generate_to_method(name: &syn::Ident, data: &Data) -> TokenStream2 {
    match *data {
        Data::Struct(DataStruct {
            fields: Fields::Named(ref fields),
            ..
        }) => {
            let expanded_fields = fields.named.iter().map(|field| {
                let name = &field.ident;

                quote! {
                    v.append(&mut solitaire::Wrap::wrap(&self.#name))
                }
            });

            quote! {
            let mut v = Vec::new();
                        #(#expanded_fields;)*
            v
                }
        }
        _ => unimplemented!(),
    }
}
