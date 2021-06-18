//! Derive macro logic for ToInstruction

use proc_macro::TokenStream;
use proc_macro2::{
    Span,
    TokenStream as TokenStream2,
};
use quote::{
    quote,
    quote_spanned,
};
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

pub fn generate_to_instruction(
    name: &syn::Ident,
    impl_generics: &syn::ImplGenerics,
    data: &Data,
) -> TokenStream2 {
    match *data {
        Data::Struct(DataStruct {
            fields: Fields::Named(ref fields),
            ..
        }) => {
            let expanded_appends = fields.named.iter().map(|field| {
                let name = &field.ident;
                let ty = &field.ty;

                quote! {
                    deps.append(&mut <#ty as solitaire::Peel>::deps());
                    account_metas.append(&mut <#ty as solitaire_client::Wrap>::wrap(&self.#name)?);
			if let Some(pair) = <#ty as solitaire_client::Wrap>::keypair(self.#name) {
			    signers.push(pair);
			}
                }
            });
            let client_struct_name =
                syn::Ident::new(&format!("{}Accounts", name.to_string()), Span::call_site());

            let client_struct_decl = generate_clientside_struct(&name, &client_struct_name, &data);

            quote! {
            /// Solitaire-generated client-side #name representation
            #[cfg(feature = "client")]
            #client_struct_decl

                /// Solitaire-generatied ToInstruction implementation
            #[cfg(feature = "client")]
                impl #impl_generics  solitaire_client::ToInstruction for #client_struct_name {
                    fn to_ix(
                self,
                program_id: solana_program::pubkey::Pubkey,
                ix_data: &[u8]) -> std::result::Result<
                (solitaire_client::Instruction, Vec<solitaire_client::Keypair>),
                            solitaire::ErrBox
                > {
            use solana_program::{pubkey::Pubkey, instruction::Instruction};
            let mut account_metas = Vec::new();
            let mut signers = Vec::new();
            let mut deps = Vec::new();

            #(#expanded_appends;)*

            // Add dependencies
            deps.dedup();
            let mut dep_ams = deps.iter().map(|v| solana_program::instruction::AccountMeta::new_readonly(*v, false)).collect();
            account_metas.append(&mut dep_ams);

            Ok((solana_program::instruction::Instruction::new_with_bytes(program_id,
                                             ix_data,
                                             account_metas), signers))

                    }

                }
                }
        }
        _ => unimplemented!(),
    }
}

pub fn generate_clientside_struct(
    name: &syn::Ident,
    client_struct_name: &syn::Ident,
    data: &Data,
) -> TokenStream2 {
    match *data {
        Data::Struct(DataStruct {
            fields: Fields::Named(ref fields),
            ..
        }) => {
            let expanded_fields = fields.named.iter().map(|field| {
                let field_name = &field.ident;

                quote! {
                    #field_name: solitaire_client::AccEntry
                }
            });

            quote! {
                        pub struct #client_struct_name {
                #(pub #expanded_fields,)*
            }
            }
        }
        _ => unimplemented!(),
    }
}
