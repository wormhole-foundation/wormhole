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
    FieldsNamed,
    GenericParam,
    Generics,
    Index,
};

pub fn generate_to_instruction(
    orig_struct_ident: &syn::Ident,
    impl_generics: &syn::ImplGenerics,
    data: &Data,
) -> TokenStream2 {
    match *data {
        Data::Struct(DataStruct {
            fields: Fields::Named(ref fields),
            ..
        }) => {
            let client_struct_ident = syn::Ident::new(
                &format!("{}Accounts", orig_struct_ident.to_string()),
                Span::call_site(),
            );
            let client_struct_decl =
                generate_clientside_struct(&orig_struct_ident, &client_struct_ident, &fields);

            let acc_metas_ident = syn::Ident::new("acc_metas", Span::call_site());
            let acc_metas_appends = generate_acc_metas_appends(&acc_metas_ident, &fields);

            let signers_ident = syn::Ident::new("signers", Span::call_site());
            let signers_appends = generate_signers_appends(&signers_ident, &fields);

            let deps_ident = syn::Ident::new("deps", Span::call_site());
            let deps_appends = generate_deps_appends(&deps_ident, &fields);

            quote! {
                /// Solitaire-generated client-side #orig_struct_ident representation
                #[cfg(feature = "client")]
		#[derive(Debug)]
                #client_struct_decl

                    /// Solitaire-generatied ToInstruction implementation
                #[cfg(feature = "client")]
		impl #impl_generics  solitaire_client::ToInstruction for #client_struct_ident {

		    fn gen_client_ix(
			&self,
			program_id: solana_program::pubkey::Pubkey,
			ix_data: &[u8]) -> std::result::Result<
			    (solitaire_client::Instruction, Vec<solitaire_client::Keypair>),
			    solitaire::ErrBox> {

			use solana_program::{pubkey::Pubkey, instruction::Instruction};
			let mut #acc_metas_ident = Vec::new();
			let mut #signers_ident = Vec::new();
			let mut #deps_ident = Vec::new();

			#acc_metas_appends
			#deps_appends
			#signers_appends

			// Add dependencies
			#deps_ident.dedup();
			let mut dep_ams = deps.iter().map(|v| solana_program::instruction::AccountMeta::new_readonly(*v, false)).collect();
			#acc_metas_ident.append(&mut dep_ams);

			Ok((solana_program::instruction::Instruction::new_with_bytes(program_id,
							ix_data,
							#acc_metas_ident), #signers_ident))

		    }

		    fn gen_client_metas(&self) -> std::result::Result <Vec<solana_program::instruction::AccountMeta>, solitaire::ErrBox> {
			let mut #acc_metas_ident = Vec::new();

			#acc_metas_appends

			Ok(#acc_metas_ident)
		    }

		    fn gen_client_signers(&self) -> Vec<solitaire_client::solana_sdk::signature::Keypair> {
			let mut #signers_ident = Vec::new();

			#signers_appends

			#signers_ident
		    }
	    }
	}
        }
        _ => unimplemented!(),
    }
}

/// Generates Wrap::wrap() calls and appends to the account metas vec
/// for the specified vec name ident and provided field data.
pub fn generate_acc_metas_appends(vec_ident: &syn::Ident, data: &FieldsNamed) -> TokenStream2 {
    let appends = data.named.iter().map(|f| {
        let f_ty = &f.ty;
        let f_ident = &f.ident;
        quote! {
        #vec_ident.append(&mut <#f_ty as solitaire_client::Wrap>::wrap(&self.#f_ident)?);
        }
    });
    quote! {
    #(#appends;)*
    }
}

/// Generates Peel::deps() calls and appends to the dependency pubkey
/// vec for the specified vec name ident and provided field data.
pub fn generate_deps_appends(vec_ident: &syn::Ident, data: &FieldsNamed) -> TokenStream2 {
    let appends = data.named.iter().map(|f| {
        let f_ty = &f.ty;
        quote! {
            #vec_ident.append(&mut <#f_ty as solitaire::Peel>::deps());
        }
    });
    quote! {
        #(#appends;)*
    }
}

/// Generates Wrap::partial_signer_keypairs() calls and appends to the
/// dependency pubkey vec for the specified vec name ident and
/// provided field data.
pub fn generate_signers_appends(vec_ident: &syn::Ident, data: &FieldsNamed) -> TokenStream2 {
    let appends = data.named.iter().map(|f| {
        let f_ty = &f.ty;
        let f_ident = &f.ident;
        quote! {
            #vec_ident.append(&mut <#f_ty as solitaire_client::Wrap>::partial_signer_keypairs(&self.#f_ident));
        }
    });
    quote! {
        #(#appends;)*
    }
}

/// Generates a client-side struct for sending relevant account pubkeys/keypairs/sysvars to RPC
pub fn generate_clientside_struct(
    orig_struct_ident: &syn::Ident,
    client_struct_ident: &syn::Ident,
    fields: &FieldsNamed,
) -> TokenStream2 {
    let expanded_fields = fields.named.iter().map(|field| {
        let field_name = &field.ident;

        quote! {
            #field_name: solitaire_client::AccEntry
        }
    });

    quote! {
        /// This Solitaire-generated account represents #orig_struct_ident off-chain on client side.
        pub struct #client_struct_ident {
            #(pub #expanded_fields,)*
        }
    }
}
