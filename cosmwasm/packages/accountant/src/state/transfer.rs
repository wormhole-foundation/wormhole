use std::{fmt, str::FromStr};

use anyhow::{ensure, Context};
use cosmwasm_schema::cw_serde;
use cosmwasm_std::{StdResult, Uint256};
use cw_storage_plus::{Key as CwKey, KeyDeserialize, Prefixer, PrimaryKey};

use crate::state::TokenAddress;

#[cw_serde]
pub struct Transfer {
    pub key: Key,
    pub data: Data,
}

#[cw_serde]
#[derive(Eq, PartialOrd, Ord, Default, Hash)]
pub struct TokenAddresses(pub Vec<TokenAddress>);

#[cw_serde]
#[derive(Eq, PartialOrd, Ord, Default, Hash)]
pub struct Key {
    // The chain id of the chain on which this transfer originated.
    emitter_chain: u16,

    // The address on the emitter chain that created this transfer.
    emitter_addresses: TokenAddresses,

    // The sequence number of the transfer.
    sequence: u64,
}

impl Key {
    pub fn new(emitter_chain: u16, emitter_addresses: TokenAddresses, sequence: u64) -> Self {
        Self {
            emitter_chain,
            emitter_addresses,
            sequence,
        }
    }

    pub fn emitter_chain(&self) -> u16 {
        self.emitter_chain
    }

    pub fn emitter_addresses(&self) -> &Vec<TokenAddress> {
        &self.emitter_addresses.0
    }

    pub fn sequence(&self) -> u64 {
        self.sequence
    }
}

impl fmt::Display for TokenAddresses {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        // write!(f, "/")?;

        for (index, address) in self.0.iter().enumerate() {
            if index > 0 {
                write!(f, "/")?;
            }
            write!(f, "{}", address)?;
        }

        write!(f, "")
    }
}

impl fmt::Display for Key {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{:05}/{}/{:016}",
            self.emitter_chain, self.emitter_addresses, self.sequence
        )
    }
}

impl FromStr for TokenAddresses {
    type Err = anyhow::Error;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        let mut items = Vec::new();

        for item_str in s.split(',') {
            let item = item_str.trim().parse()?;
            items.push(item);
        }

        Ok(TokenAddresses(items))
    }
}

impl FromStr for Key {
    type Err = anyhow::Error;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        let mut components = s.split('/');
        let emitter_chain = components
            .next()
            .map(str::parse)
            .transpose()
            .context("failed to parse emitter chain")?
            .context("emitter chain missing")?;
        let emitter_addresses = components
            .next()
            .map(str::parse)
            .transpose()
            .context("failed to parse emitter address")?
            .context("emitter address missing")?;
        let sequence = components
            .next()
            .map(str::parse)
            .transpose()
            .context("failed to parse sequence")?
            .context("sequence missing")?;

        ensure!(
            components.next().is_none(),
            "unexpected trailing input data"
        );

        Ok(Key {
            emitter_chain,
            emitter_addresses,
            sequence,
        })
    }
}

impl KeyDeserialize for TokenAddresses {
    type Output = Self;

    fn from_vec(v: Vec<u8>) -> StdResult<Self::Output> {
        let chunks = v.chunks(32).collect::<Vec<_>>();
        let mut token_addresses = Vec::new();
        for chunk in chunks.iter() {
            let addr = <TokenAddress>::from_vec(chunk.to_vec());
            token_addresses.push(addr.unwrap())
        }

        Ok(TokenAddresses(token_addresses))
    }
}

impl KeyDeserialize for Key {
    type Output = Self;

    fn from_vec(v: Vec<u8>) -> StdResult<Self::Output> {
        <(u16, TokenAddresses, u64)>::from_vec(v).map(
            |(emitter_chain, emitter_addresses, sequence)| Key {
                emitter_chain,
                emitter_addresses,
                sequence,
            },
        )
    }
}

impl<'a> Prefixer<'a> for TokenAddresses {
    fn prefix(&self) -> Vec<CwKey> {
        self.0.iter().flat_map(|address| address.prefix()).collect()
    }
}

impl<'a> PrimaryKey<'a> for Key {
    type Prefix = (u16, TokenAddresses);
    type SubPrefix = u16;
    type Suffix = u64;
    type SuperSuffix = (TokenAddresses, u64);

    fn key(&self) -> Vec<CwKey> {
        self.emitter_chain
            .key()
            .into_iter()
            .chain(
                self.emitter_addresses
                    .0
                    .iter()
                    .flat_map(|a| a.key())
                    .collect::<Vec<_>>(),
            )
            .chain(self.sequence.key())
            .collect()
    }
}

#[cw_serde]
pub struct Data {
    // The amount to be transferred.
    pub amount: Uint256,

    // The id of the native chain of the token.
    pub token_chain: u16,

    // The address of the token on its native chain.
    pub token_address: TokenAddress,

    // The chain id where the tokens are being sent.
    pub recipient_chain: u16,
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn key_display() {
        let addr = TokenAddress::new([
            0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 62, 225, 139, 34, 20, 175, 249, 112, 0, 217, 116,
            207, 100, 126, 124, 52, 126, 143, 165, 133,
        ]);
        let addr_1 = TokenAddress::new([
            0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 62, 225, 139, 34, 20, 175, 249, 112, 0, 217, 116,
            207, 100, 126, 124, 52, 126, 143, 165, 133,
        ]);
        let mut addrs = Vec::new();
        addrs.push(addr);
        addrs.push(addr_1);
        let k = Key::new(2, TokenAddresses(addrs), 254278);
        let expected = "00002/0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585/0000010000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585/0000000000254278";
        assert_eq!(expected, k.to_string());
    }
}
