use std::ops::{Deref, DerefMut};

use cosmwasm_schema::cw_serde;
use cosmwasm_std::{StdResult, Uint256};
use cw_storage_plus::{Key as CwKey, KeyDeserialize, PrimaryKey};

use crate::state::TokenAddress;

#[cw_serde]
pub struct Account {
    pub key: Key,

    // The current balance of the account.
    pub balance: Balance,
}

impl Account {
    pub fn lock_or_burn(&mut self, amt: Uint256) -> StdResult<()> {
        if self.key.chain_id == self.key.token_chain {
            self.balance.0 = self.balance.checked_add(amt)?;
        } else {
            self.balance.0 = self.balance.checked_sub(amt)?;
        }

        Ok(())
    }

    pub fn unlock_or_mint(&mut self, amt: Uint256) -> StdResult<()> {
        if self.key.chain_id == self.key.token_chain {
            self.balance.0 = self.balance.checked_sub(amt)?;
        } else {
            self.balance.0 = self.balance.checked_add(amt)?;
        }

        Ok(())
    }
}

#[cw_serde]
#[derive(Eq, PartialOrd, Ord)]
pub struct Key {
    // The chain id of the chain to which this account belongs.
    chain_id: u16,
    // The chain id of the native chain for the token associated with this account.
    token_chain: u16,
    // The address of the token associated with this account on its native chain.
    token_address: TokenAddress,
}

impl Key {
    pub fn new(chain_id: u16, token_chain: u16, token_address: TokenAddress) -> Self {
        Self {
            chain_id,
            token_chain,
            token_address,
        }
    }

    pub fn chain_id(&self) -> u16 {
        self.chain_id
    }

    pub fn token_chain(&self) -> u16 {
        self.token_chain
    }

    pub fn token_address(&self) -> &TokenAddress {
        &self.token_address
    }
}

impl KeyDeserialize for Key {
    type Output = Self;

    fn from_vec(v: Vec<u8>) -> StdResult<Self::Output> {
        <(u16, u16, TokenAddress)>::from_vec(v).map(|(chain_id, token_chain, token_address)| Key {
            chain_id,
            token_chain,
            token_address,
        })
    }
}

impl<'a> PrimaryKey<'a> for Key {
    type Prefix = (u16, u16);
    type SubPrefix = u16;
    type Suffix = TokenAddress;
    type SuperSuffix = (u16, TokenAddress);

    fn key(&self) -> Vec<CwKey> {
        self.chain_id
            .key()
            .into_iter()
            .chain(self.token_chain.key())
            .chain(self.token_address.key())
            .collect()
    }
}

#[cw_serde]
pub struct Balance(Uint256);

impl Balance {
    pub const fn new(v: Uint256) -> Balance {
        Balance(v)
    }

    pub const fn zero() -> Balance {
        Balance(Uint256::zero())
    }
}

impl Deref for Balance {
    type Target = Uint256;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for Balance {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl AsRef<Uint256> for Balance {
    fn as_ref(&self) -> &Uint256 {
        &self.0
    }
}

impl AsMut<Uint256> for Balance {
    fn as_mut(&mut self) -> &mut Uint256 {
        &mut self.0
    }
}

impl From<Uint256> for Balance {
    fn from(v: Uint256) -> Self {
        Balance(v)
    }
}

impl From<Balance> for Uint256 {
    fn from(b: Balance) -> Self {
        b.0
    }
}

#[cfg(test)]
mod test {
    use cosmwasm_std::StdError;

    use super::*;

    #[test]
    fn native_lock() {
        let mut acc = Account {
            key: Key {
                chain_id: 0xbae2,
                token_chain: 0xbae2,
                token_address: TokenAddress::new([
                    0x62, 0x4e, 0x8d, 0xc6, 0xe0, 0xfe, 0x16, 0xe2, 0x59, 0x6e, 0xcf, 0x9f, 0x90,
                    0x0e, 0xd9, 0x5f, 0x4e, 0x6d, 0x26, 0xea, 0xf1, 0x9e, 0xe3, 0xe2, 0x88, 0x63,
                    0x60, 0xff, 0xc4, 0x1b, 0xfb, 0x61,
                ]),
            },

            balance: Balance(500u128.into()),
        };

        acc.lock_or_burn(200u128.into()).unwrap();

        assert_eq!(acc.balance.0, Uint256::from(700u128));
    }

    #[test]
    fn native_lock_overflow() {
        let mut acc = Account {
            key: Key {
                chain_id: 0xbae2,
                token_chain: 0xbae2,
                token_address: TokenAddress::new([
                    0x62, 0x4e, 0x8d, 0xc6, 0xe0, 0xfe, 0x16, 0xe2, 0x59, 0x6e, 0xcf, 0x9f, 0x90,
                    0x0e, 0xd9, 0x5f, 0x4e, 0x6d, 0x26, 0xea, 0xf1, 0x9e, 0xe3, 0xe2, 0x88, 0x63,
                    0x60, 0xff, 0xc4, 0x1b, 0xfb, 0x61,
                ]),
            },

            balance: Balance(Uint256::MAX),
        };

        let e = acc.lock_or_burn(200u128.into()).unwrap_err();

        assert!(matches!(e, StdError::Overflow { .. }))
    }

    #[test]
    fn native_unlock() {
        let mut acc = Account {
            key: Key {
                chain_id: 0xbae2,
                token_chain: 0xbae2,
                token_address: TokenAddress::new([
                    0x62, 0x4e, 0x8d, 0xc6, 0xe0, 0xfe, 0x16, 0xe2, 0x59, 0x6e, 0xcf, 0x9f, 0x90,
                    0x0e, 0xd9, 0x5f, 0x4e, 0x6d, 0x26, 0xea, 0xf1, 0x9e, 0xe3, 0xe2, 0x88, 0x63,
                    0x60, 0xff, 0xc4, 0x1b, 0xfb, 0x61,
                ]),
            },

            balance: Balance(500u128.into()),
        };

        acc.unlock_or_mint(200u128.into()).unwrap();

        assert_eq!(acc.balance.0, Uint256::from(300u128));
    }

    #[test]
    fn native_unlock_underflow() {
        let mut acc = Account {
            key: Key {
                chain_id: 0xbae2,
                token_chain: 0xbae2,
                token_address: TokenAddress::new([
                    0x62, 0x4e, 0x8d, 0xc6, 0xe0, 0xfe, 0x16, 0xe2, 0x59, 0x6e, 0xcf, 0x9f, 0x90,
                    0x0e, 0xd9, 0x5f, 0x4e, 0x6d, 0x26, 0xea, 0xf1, 0x9e, 0xe3, 0xe2, 0x88, 0x63,
                    0x60, 0xff, 0xc4, 0x1b, 0xfb, 0x61,
                ]),
            },

            balance: Balance(Uint256::zero()),
        };

        let e = acc.unlock_or_mint(200u128.into()).unwrap_err();

        assert!(matches!(e, StdError::Overflow { .. }))
    }

    #[test]
    fn wrapped_burn() {
        let mut acc = Account {
            key: Key {
                chain_id: 0xcae8,
                token_chain: 0xbae2,
                token_address: TokenAddress::new([
                    0x62, 0x4e, 0x8d, 0xc6, 0xe0, 0xfe, 0x16, 0xe2, 0x59, 0x6e, 0xcf, 0x9f, 0x90,
                    0x0e, 0xd9, 0x5f, 0x4e, 0x6d, 0x26, 0xea, 0xf1, 0x9e, 0xe3, 0xe2, 0x88, 0x63,
                    0x60, 0xff, 0xc4, 0x1b, 0xfb, 0x61,
                ]),
            },

            balance: Balance(500u128.into()),
        };

        acc.lock_or_burn(200u128.into()).unwrap();

        assert_eq!(acc.balance.0, Uint256::from(300u128));
    }

    #[test]
    fn wrapped_burn_underflow() {
        let mut acc = Account {
            key: Key {
                chain_id: 0xcae8,
                token_chain: 0xbae2,
                token_address: TokenAddress::new([
                    0x62, 0x4e, 0x8d, 0xc6, 0xe0, 0xfe, 0x16, 0xe2, 0x59, 0x6e, 0xcf, 0x9f, 0x90,
                    0x0e, 0xd9, 0x5f, 0x4e, 0x6d, 0x26, 0xea, 0xf1, 0x9e, 0xe3, 0xe2, 0x88, 0x63,
                    0x60, 0xff, 0xc4, 0x1b, 0xfb, 0x61,
                ]),
            },

            balance: Balance(Uint256::zero()),
        };

        let e = acc.lock_or_burn(200u128.into()).unwrap_err();

        assert!(matches!(e, StdError::Overflow { .. }))
    }

    #[test]
    fn wrapped_mint() {
        let mut acc = Account {
            key: Key {
                chain_id: 0xcae8,
                token_chain: 0xbae2,
                token_address: TokenAddress::new([
                    0x62, 0x4e, 0x8d, 0xc6, 0xe0, 0xfe, 0x16, 0xe2, 0x59, 0x6e, 0xcf, 0x9f, 0x90,
                    0x0e, 0xd9, 0x5f, 0x4e, 0x6d, 0x26, 0xea, 0xf1, 0x9e, 0xe3, 0xe2, 0x88, 0x63,
                    0x60, 0xff, 0xc4, 0x1b, 0xfb, 0x61,
                ]),
            },

            balance: Balance(500u128.into()),
        };

        acc.unlock_or_mint(200u128.into()).unwrap();

        assert_eq!(acc.balance.0, Uint256::from(700u128));
    }

    #[test]
    fn wrapped_mint_overflow() {
        let mut acc = Account {
            key: Key {
                chain_id: 0xcae8,
                token_chain: 0xbae2,
                token_address: TokenAddress::new([
                    0x62, 0x4e, 0x8d, 0xc6, 0xe0, 0xfe, 0x16, 0xe2, 0x59, 0x6e, 0xcf, 0x9f, 0x90,
                    0x0e, 0xd9, 0x5f, 0x4e, 0x6d, 0x26, 0xea, 0xf1, 0x9e, 0xe3, 0xe2, 0x88, 0x63,
                    0x60, 0xff, 0xc4, 0x1b, 0xfb, 0x61,
                ]),
            },

            balance: Balance(Uint256::MAX),
        };

        let e = acc.unlock_or_mint(200u128.into()).unwrap_err();

        assert!(matches!(e, StdError::Overflow { .. }))
    }
}
