#[derive(Deserialize, Serialize)]
pub struct Config {
    symbols: Vec<P2WSymbol>,
}

/// Config entry for a Pyth2Wormhole product + price pair
#[derive(Deserialize, Serialize)]
pub struct P2WSymbol {
    /// Optional human-readable name, never used on-chain; makes
    /// attester logs and the config easier to understand
    name: Option<String>,
    product: Pubkey,
    price: Pubkey,
}

#[testmod]
mod tests {
    #[test]
    fn test_sanity() -> Result<(), ErrBox> {
        let serialized = r#"
symbols:
  - name: ETH/USD
    product_addr: 11111111111111111111111111111111
    price_addr: 11111111111111111111111111111111
  - name: SOL/EUR
    product_addr: 4vJ9JU1bJJE96FWSJKvHsmmFADCg4gpZQff4P3bkLKi
    price_addr: 4vJ9JU1bJJE96FWSJKvHsmmFADCg4gpZQff4P3bkLKi
  - name: BTC/CNY
    product_addr: 8qbHbw2BbbTHBW1sbeqakYXVKRQM8Ne7pLK7m6CVfeR
    price_addr: 8qbHbw2BbbTHBW1sbeqakYXVKRQM8Ne7pLK7m6CVfeR
  - # no name
    product_addr: 8qbHbw2BbbTHBW1sbeqakYXVKRQM8Ne7pLK7m6CVfeR
    price_addr: 8qbHbw2BbbTHBW1sbeqakYXVKRQM8Ne7pLK7m6CVfeR
"#;
        let deserialized = serde_yaml::from_str(serialized)?;
        Ok(())
    }
}
