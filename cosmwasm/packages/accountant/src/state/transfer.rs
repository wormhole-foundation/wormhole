use std::{fmt, str::FromStr};

use anyhow::{ensure, Context};
use cosmwasm_schema::cw_serde;
use cosmwasm_std::{StdResult, Uint256};
use cw_storage_plus::{Key as CwKey, KeyDeserialize, PrimaryKey};

use crate::state::TokenAddress;

#[cw_serde]
pub struct Transfer {
    pub key: Key,
    pub data: Data,
}

#[cw_serde]
#[derive(Eq, PartialOrd, Ord, Default, Hash)]
pub struct Key {
    // The chain id of the chain on which this transfer originated.
    emitter_chain: u16,

    // The address on the emitter chain that created this transfer.
    emitter_address: TokenAddress,

    // The sequence number of the transfer.
    sequence: u64,
}

impl Key {
    pub fn new(emitter_chain: u16, emitter_address: TokenAddress, sequence: u64) -> Self {
        Self {
            emitter_chain,
            emitter_address,
            sequence,
        }
    }

    pub fn emitter_chain(&self) -> u16 {
        self.emitter_chain
    }

    pub fn emitter_address(&self) -> &TokenAddress {
        &self.emitter_address
    }

    pub fn sequence(&self) -> u64 {
        self.sequence
    }
}

impl fmt::Display for Key {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{:05}/{}/{:016}",
            self.emitter_chain, self.emitter_address, self.sequence
        )
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
        let emitter_address = components
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
            emitter_address,
            sequence,
        })
    }
}

impl KeyDeserialize for Key {
    type Output = Self;

    fn from_vec(v: Vec<u8>) -> StdResult<Self::Output> {
        <(u16, TokenAddress, u64)>::from_vec(v).map(|(emitter_chain, emitter_address, sequence)| {
            Key {
                emitter_chain,
                emitter_address,
                sequence,
            }
        })
    }
}

impl<'a> PrimaryKey<'a> for Key {
    type Prefix = (u16, TokenAddress);
    type SubPrefix = u16;
    type Suffix = u64;
    type SuperSuffix = (TokenAddress, u64);

    fn key(&self) -> Vec<CwKey> {
        self.emitter_chain
            .key()
            .into_iter()
            .chain(self.emitter_address.key())
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
        let k = Key::new(2, addr, 254278);
        let expected = "00002/0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585/0000000000254278";
        assert_eq!(expected, k.to_string());
    }
}
