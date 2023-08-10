use cosmwasm_schema::cw_serde;
use cosmwasm_std::{CosmosMsg, CustomMsg};

/// A top-level Custom message for the token factory.
/// It is embedded like this to easily allow adding other variants that are custom
/// to your chain, or other "standardized" extensions along side it.
#[cw_serde]
pub enum TokenFactoryMsg {
    Token(TokenMsg),
}

/// Special messages to be supported by any chain that supports token_factory
#[cw_serde]
pub enum TokenMsg {
    /// CreateDenom creates a new factory denom, of denomination:
    /// factory/{creating contract bech32 address}/{Subdenom}
    /// Subdenom can be of length at most 44 characters, in [0-9a-zA-Z./]
    /// Empty subdenoms are valid.
    /// The (creating contract address, subdenom) pair must be unique.
    /// The created denom's admin is the creating contract address,
    /// but this admin can be changed using the UpdateAdmin binding.
    ///
    /// If you set an initial metadata here, this is equivalent
    /// to calling SetMetadata directly on the returned denom.
    CreateDenom {
        subdenom: String,
        metadata: Option<Metadata>,
    },
    /// ChangeAdmin changes the admin for a factory denom.
    /// Can only be called by the current contract admin.
    /// If the NewAdminAddress is empty, the denom will have no admin.
    ChangeAdmin {
        denom: String,
        new_admin_address: String,
    },
    /// Contracts can mint native tokens for an existing factory denom
    /// that they are the admin of.
    MintTokens {
        denom: String,
        amount: u128,
        mint_to_address: String,
    },
    /// Contracts can burn native tokens for an existing factory denom
    /// that they are the admin of.
    BurnTokens {
        denom: String,
        amount: u128,
        burn_from_address: String,
    },
    /// Contracts can force transfer tokens for an existing factory denom
    /// that they are the admin of.    
    ForceTransfer {
        denom: String,
        amount: u128,
        from_address: String,
        to_address: String,
    },
    SetMetadata {
        denom: String,
        metadata: Metadata,
    },
}

impl TokenMsg {
    pub fn mint_contract_tokens(denom: String, amount: u128, mint_to_address: String) -> Self {
        TokenMsg::MintTokens {
            denom,
            amount,
            mint_to_address,
        }
    }

    pub fn burn_contract_tokens(denom: String, amount: u128, burn_from_address: String) -> Self {
        TokenMsg::BurnTokens {
            denom,
            amount,
            burn_from_address,
        }
    }

    pub fn force_transfer_tokens(
        denom: String,
        amount: u128,
        from_address: String,
        to_address: String,
    ) -> Self {
        TokenMsg::ForceTransfer {
            denom,
            amount,
            from_address,
            to_address,
        }
    }
}

impl From<TokenMsg> for CosmosMsg<TokenFactoryMsg> {
    fn from(msg: TokenMsg) -> CosmosMsg<TokenFactoryMsg> {
        CosmosMsg::Custom(TokenFactoryMsg::Token(msg))
    }
}

impl CustomMsg for TokenFactoryMsg {}

/// This maps to cosmos.bank.v1beta1.Metadata protobuf struct
#[cw_serde]
pub struct Metadata {
    pub description: Option<String>,
    /// denom_units represents the list of DenomUnit's for a given coin
    pub denom_units: Vec<DenomUnit>,
    /// base represents the base denom (should be the DenomUnit with exponent = 0).
    pub base: Option<String>,
    /// display indicates the suggested denom that should be displayed in clients.
    pub display: Option<String>,
    /// name defines the name of the token (eg: Cosmos Atom)
    pub name: Option<String>,
    /// symbol is the token symbol usually shown on exchanges (eg: ATOM). This can
    /// be the same as the display.
    pub symbol: Option<String>,
}

/// This maps to cosmos.bank.v1beta1.DenomUnit protobuf struct
#[cw_serde]
pub struct DenomUnit {
    /// denom represents the string name of the given denom unit (e.g uatom).
    pub denom: String,
    /// exponent represents power of 10 exponent that one must
    /// raise the base_denom to in order to equal the given DenomUnit's denom
    /// 1 denom = 1^exponent base_denom
    /// (e.g. with a base_denom of uatom, one can create a DenomUnit of 'atom' with
    /// exponent = 6, thus: 1 atom = 10^6 uatom).
    pub exponent: u32,
    /// aliases is a list of string aliases for the given denom
    pub aliases: Vec<String>,
}
