use anchor_lang::prelude::*;
use serde::{Deserialize, Serialize};
use wormhole_solana_common::{legacy_account, LegacyDiscriminator};

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct MetadataUri {
    chain: u16,
    address: String,
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
    pub fn to_uri(&self) -> serde_json::Result<String> {
        let mut uri = serde_json::to_string_pretty(&MetadataUri {
            chain: self.token_chain,
            address: hex::encode(self.token_address),
            native_decimals: self.native_decimals,
        })?;

        // Unlikely to happen, but truncate the URI if it's too long.
        uri.truncate(mpl_token_metadata::state::MAX_URI_LENGTH);

        Ok(uri)
    }
}

impl LegacyDiscriminator<0> for WrappedAsset {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
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
  "chain": 420,
  "address": "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
  "nativeDecimals": 18
}"#;

        assert_eq!(asset.to_uri().unwrap(), expected);
    }
}
