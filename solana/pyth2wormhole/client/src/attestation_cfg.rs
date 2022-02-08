use std::str::FromStr;

use serde::{
    de::Error,
    Deserialize,
    Deserializer,
    Serialize,
    Serializer,
};
use solana_program::pubkey::Pubkey;
use solitaire::ErrBox;

/// Pyth2wormhole config specific to attestation requests
#[derive(Debug, Deserialize, Serialize, PartialEq, Eq)]
pub struct AttestationConfig {
    pub symbols: Vec<P2WSymbol>,
}

/// Config entry for a Pyth product + price pair
#[derive(Debug, Deserialize, Serialize, PartialEq, Eq)]
pub struct P2WSymbol {
    /// User-defined human-readable name
    pub name: Option<String>,

    #[serde(
        deserialize_with = "pubkey_string_de",
        serialize_with = "pubkey_string_ser"
    )]
    pub product_addr: Pubkey,
    #[serde(
        deserialize_with = "pubkey_string_de",
        serialize_with = "pubkey_string_ser"
    )]
    pub price_addr: Pubkey,
}

// Helper methods for strinigified SOL addresses

fn pubkey_string_ser<S>(k: &Pubkey, ser: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    ser.serialize_str(&k.to_string())
}

fn pubkey_string_de<'de, D>(de: D) -> Result<Pubkey, D::Error>
where
    D: Deserializer<'de>,
{
    let pubkey_string = String::deserialize(de)?;
    let pubkey = Pubkey::from_str(&pubkey_string).map_err(D::Error::custom)?;
    Ok(pubkey)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sanity() -> Result<(), ErrBox> {
        let initial = AttestationConfig {
            symbols: vec![
                P2WSymbol {
                    name: Some("ETH/USD".to_owned()),
                    product_addr: Default::default(),
                    price_addr: Default::default(),
                },
                P2WSymbol {
                    name: None,
                    product_addr: Pubkey::new(&[42u8; 32]),
                    price_addr: Default::default(),
                },
            ],
        };

        let serialized = serde_yaml::to_string(&initial)?;
        eprintln!("Serialized:\n{}", serialized);

        let deserialized: AttestationConfig = serde_yaml::from_str(&serialized)?;

        assert_eq!(initial, deserialized);
        Ok(())
    }
}
