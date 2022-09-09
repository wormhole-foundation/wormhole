use cosmwasm_std::{
    Addr,
    StdError,
    StdResult,
    Storage,
};

use schemars::JsonSchema;
use serde::{
    Deserialize,
    Serialize,
};
use sha3::{
    Digest,
    Keccak256,
};

use crate::state::{
    bank_token_hashes,
    bank_token_hashes_read,
    config_read,
    native_c20_hashes,
    native_c20_hashes_read,
};

/// Represent the external view of a token address.
/// This is the value that goes into the VAA.
///
/// When given an external 32 byte address, there are 3 options:
/// I. This is a token native to this chain
///     a. it's a token managed by the Bank cosmos module
///     (e.g. the staking denom "uluna" on Terra)
///     b. it's a CW20 token
/// II. This is a token address from another chain
///
/// Thus, interpreting an external token id requires knowing whether the token
/// in question originates from this chain, or another chain. This information
/// will always be available from the context.
///
/// I. //////////////////////////////////////////////////////////////////////////
///
/// In the first case (native tokens), the layout of is the following:
///
///  | 1 byte |                          31 bytes                               |
///  +--------+-----------------------------------------------------------------+
///  | MARKER |                           HASH                                  |
///  +--------+-----------------------------------------------------------------+
///
/// The left-most byte (MARKER) tells us whether it's a Bank token (1), or a CW20 (0).
/// Since denom names can be arbitarily long, and CW20 addresses are 32 byes, we
/// cannot directly encode them into the remaining 31 bytes. Instead, we hash
/// the data (either the denom or the CW20 address), and put the last 31 bytes
/// of the hash into the address (HASH). In particular, this choice reduces the
/// space of the hash function by 8 bits, but assuming the hash is resistant to
/// differential attacks, we consider giving up on these 8 bits safe.
///
/// In order to be able to recover the denom and the contract address later, we
/// store a mapping from these 32 bytes (MARKER+HASH) to denoms and CW20
/// addresses (c.f. [`native_cw20_hashes`] & [`bank_token_hashes`] in state.rs)
///
/// II. /////////////////////////////////////////////////////////////////////////
///
/// In the second case (foreign tokens), the whole 32 bytes correspond to the
/// external token address. In this case, the corresponding token will be a
/// wrapped asset, whose address is stored in storage as a mapping (c.f.
/// [`wrapped_asset`] in state.rs)
///
///    (chain_id, external_id) => wrapped_asset_address
///
/// For internal consumption of these addresses, we first convert them to
/// [`TokenId`] (see below).
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[repr(transparent)]
pub struct ExternalTokenId {
    bytes: [u8; 32],
}

/// The intention is that [`ExternalTokenId`] should always be converted to/from
/// [`TokenId`] through the functions defined in this module. This is why its
/// internal contents are private. The following functions are called
/// [`serialize`] and [`deserialize`] to signify that they should only be used
/// when converting back and forth the wire format.
impl ExternalTokenId {
    pub fn serialize(&self) -> [u8; 32] {
        self.bytes
    }

    pub fn deserialize(data: [u8; 32]) -> Self {
        Self { bytes: data }
    }
}

impl ExternalTokenId {
    pub fn to_token_id(&self, storage: &dyn Storage, origin_chain: u16) -> StdResult<TokenId> {
        let state = config_read(storage).load()?;
        if origin_chain == state.chain_id {
            let marker_byte = self.bytes[0];
            match marker_byte {
                1 => {
                    let denom = bank_token_hashes_read(storage).load(&self.bytes)?;
                    Ok(TokenId::Bank { denom })
                }
                0 => {
                    let human_address = native_c20_hashes_read(storage).load(&self.bytes)?;
                    Ok(TokenId::Contract(ContractId::NativeCW20 {
                        contract_address: human_address,
                    }))
                }
                b => Err(StdError::generic_err(format!("Unknown marker byte: {}", b))),
            }
        } else {
            Ok(TokenId::Contract(ContractId::ForeignToken {
                chain_id: origin_chain,
                foreign_address: self.bytes,
            }))
        }
    }

    pub fn from_native_cw20(contract_address: &Addr) -> StdResult<ExternalTokenId> {
        let mut hash = hash(contract_address.as_bytes());
        // override first byte with marker byte
        hash[0] = 0;
        Ok(ExternalTokenId { bytes: hash })
    }

    pub fn from_foreign_token(foreign_address: [u8; 32]) -> ExternalTokenId {
        ExternalTokenId {
            bytes: foreign_address,
        }
    }

    pub fn from_bank_token(denom: &String) -> StdResult<ExternalTokenId> {
        let mut hash = hash(&denom.as_bytes());
        // override first byte with marker byte
        hash[0] = 1;
        Ok(ExternalTokenId { bytes: hash })
    }

    pub fn from_token_id(token_id: &TokenId) -> StdResult<ExternalTokenId> {
        match token_id {
            TokenId::Bank { denom } => Self::from_bank_token(denom),
            TokenId::Contract(contract) => match contract {
                ContractId::NativeCW20 { contract_address } => {
                    Self::from_native_cw20(contract_address)
                }
                ContractId::ForeignToken {
                    chain_id: _,
                    foreign_address,
                } => Ok(Self::from_foreign_token(*foreign_address)),
            },
        }
    }
}

/// Internal view of an address. This type is similar to [`AssetInfo`], but more
/// granular. We do differentiate between bank tokens and CW20 tokens, but in
/// the latter case, we further differentiate between native CW20s and wrapped
/// CW20s (see [`ContractId`]).
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum TokenId {
    Bank { denom: String },
    Contract(ContractId),
}

impl TokenId {
    /// Given a [`TokenId`], we can always directly construct an
    /// [`ExternalTokenId`], but the converse is not true, since the
    /// construction of external ids involves hashing, and is thus irreversible.
    /// To maintain the bijection (modulo hash collisions), we store the hash
    /// preimages in storage, so given an external id, the necessary information
    /// can always be queried to reconstruct the token id.
    /// This information must be stored when the token id is first converted to
    /// an external id, i.e. when an attestation is generated for the token.
    pub fn store(&self, storage: &mut dyn Storage) -> StdResult<ExternalTokenId> {
        let external_id = ExternalTokenId::from_token_id(self)?;
        match self {
            TokenId::Bank { denom } => bank_token_hashes(storage).save(&external_id.bytes, denom),
            TokenId::Contract(contract) => match contract {
                ContractId::NativeCW20 { contract_address } => {
                    native_c20_hashes(storage).save(&external_id.bytes, &contract_address)
                }
                ContractId::ForeignToken {
                    chain_id: _,
                    foreign_address: _,
                } => Err(StdError::generic_err(
                    "Foreign tokens are not stored in storage",
                )),
            },
        }?;
        Ok(external_id)
    }
}

/// A contract id is either a native cw20 address, or a foreign token. The
/// reason we represent the foreign address here instead of storing the wrapped
/// CW20 contract's address directly is that the wrapped asset might not be
/// deployed yet.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum ContractId {
    NativeCW20 {
        contract_address: Addr,
    },
    /// A wrapped token might not exist yet.
    ForeignToken {
        chain_id: u16,
        foreign_address: [u8; 32],
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[repr(transparent)]
pub struct WrappedCW20 {
    pub human_address: Addr,
}

impl WrappedCW20 {
    pub fn into_string(self) -> String {
        self.human_address.into_string()
    }
}

fn hash(bytes: &[u8]) -> [u8; 32] {
    let mut hasher = Keccak256::new();
    hasher.update(bytes);
    hasher.finalize().into()
}
