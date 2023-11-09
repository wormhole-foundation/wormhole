use anchor_lang::prelude::*;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct MetadataUri {
    wormhole_chain_id: u16,
    canonical_address: String,
    native_decimals: u8,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct LegacyWrappedAsset {
    pub token_chain: u16,
    pub token_address: [u8; 32],
    pub native_decimals: u8,
}

impl core_bridge_program::sdk::legacy::LegacyAccount for LegacyWrappedAsset {
    const DISCRIMINATOR: &'static [u8] = &[];

    fn program_id() -> Pubkey {
        crate::ID
    }
}

impl LegacyWrappedAsset {
    pub const SEED_PREFIX: &'static [u8] = b"meta";

    pub fn to_uri(&self) -> String {
        let mut uri = serde_json::to_string_pretty(&MetadataUri {
            wormhole_chain_id: self.token_chain,
            canonical_address: format!("0x{}", hex::encode(self.token_address)),
            native_decimals: self.native_decimals,
        })
        .expect("serialization should not fail");

        // Unlikely to happen, but truncate the URI if it's too long.
        uri.truncate(mpl_token_metadata::state::MAX_URI_LENGTH);

        uri
    }
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct WrappedAsset {
    pub legacy: LegacyWrappedAsset,
    pub last_updated_sequence: u64,
}

impl std::ops::Deref for WrappedAsset {
    type Target = LegacyWrappedAsset;

    fn deref(&self) -> &Self::Target {
        &self.legacy
    }
}

impl WrappedAsset {
    pub const SEED_PREFIX: &'static [u8] = LegacyWrappedAsset::SEED_PREFIX;
}

impl core_bridge_program::sdk::legacy::LegacyAccount for WrappedAsset {
    const DISCRIMINATOR: &'static [u8] = LegacyWrappedAsset::DISCRIMINATOR;

    fn program_id() -> Pubkey {
        crate::ID
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn to_uri() {
        let asset = WrappedAsset {
            legacy: LegacyWrappedAsset {
                token_chain: 420,
                token_address: [
                    222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239,
                    222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239,
                ],
                native_decimals: 18,
            },
            last_updated_sequence: 69,
        };

        let expected = r#"{
  "wormholeChainId": 420,
  "canonicalAddress": "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
  "nativeDecimals": 18
}"#;

        assert_eq!(asset.to_uri(), expected);
    }
}
