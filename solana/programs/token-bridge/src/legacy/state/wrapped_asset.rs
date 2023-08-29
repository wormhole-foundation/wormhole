use anchor_lang::prelude::*;
use serde::{Deserialize, Serialize};
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, SeedPrefix};

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct MetadataUri {
    wormhole_chain_id: u16,
    canonical_address: String,
    native_decimals: u8,
}

#[legacy_account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct WrappedAsset {
    pub token_chain: u16,
    pub token_address: [u8; 32],
    pub native_decimals: u8,
}

impl WrappedAsset {
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

impl LegacyDiscriminator<0> for WrappedAsset {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl SeedPrefix for WrappedAsset {
    const SEED_PREFIX: &'static [u8] = b"meta";
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn to_uri() {
        let asset = WrappedAsset {
            token_chain: 420,
            token_address: [
                222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239,
                222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239, 222, 173, 190, 239,
            ],
            native_decimals: 18,
        };

        let expected = r#"{
  "wormholeChainId": 420,
  "canonicalAddress": "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
  "nativeDecimals": 18
}"#;

        assert_eq!(asset.to_uri(), expected);
    }
}
