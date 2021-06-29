use std::{mem};

use pyth_client::{CorpAction, Price, PriceStatus, PriceType};
use solana_program::{program_error::ProgramError, pubkey::Pubkey};
use solitaire::{Result as SoliResult, SolitaireError};

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct PriceAttestation {
    pub product: Pubkey,
    pub price_type: PriceType,
    pub price: i64,
    pub expo: i32,
    pub confidence_interval: u64,
    pub status: PriceStatus,
    pub corp_act: CorpAction,
}

impl PriceAttestation {
    pub fn from_bytes(value: &[u8]) -> Result<Self, SolitaireError> {
        let price = parse_pyth_price(value)?;

        Ok(PriceAttestation {
            product: Pubkey::new(&price.prod.val[..]),
            price_type: price.ptype,
            price: price.agg.price,
            expo: price.expo,
            confidence_interval: price.agg.conf,
            status: price.agg.status,
            corp_act: price.agg.corp_act,
        })
    }
}

/// Deserializes Price from raw bytes
fn parse_pyth_price(price_data: &[u8]) -> SoliResult<Price> {
    if price_data.len() != mem::size_of::<Price>() {
        return Err(ProgramError::InvalidAccountData.into());
    }
    let price_account = pyth_client::cast::<Price>(price_data);

    Ok(price_account.clone())
}

#[cfg(test)]
mod tests {
    use super::*;
    use pyth_client::{AccKey, AccountType, PriceComp, PriceInfo};

    macro_rules! empty_acckey {
        () => {
            AccKey { val: [0u8; 32] }
        };
    }

    macro_rules! empty_priceinfo {
        () => {
            PriceInfo {
                price: 0,
                conf: 0,
                status: PriceStatus::Unknown,
                corp_act: CorpAction::NoCorpAct,
                pub_slot: 0,
            }
        };
    }

    macro_rules! empty_pricecomp {
        () => {
            PriceComp {
                publisher: empty_acckey!(),
                agg: empty_priceinfo!(),
                latest: empty_priceinfo!(),
            }
        };
    }

    macro_rules! empty_price {
        () => {
            Price {
                magic: pyth_client::MAGIC,
                ver: pyth_client::VERSION,
                atype: AccountType::Price as u32,
                size: 0,
                ptype: PriceType::Price,
                expo: 0,
                num: 0,
                unused: 0,
                curr_slot: 0,
                valid_slot: 0,
                twap: 0,
                avol: 0,
                drv0: 0,
                drv1: 0,
                drv2: 0,
                drv3: 0,
                drv4: 0,
                drv5: 0,
                prod: empty_acckey!(),
                next: empty_acckey!(),
                agg_pub: empty_acckey!(),
                agg: empty_priceinfo!(),
                // A nice macro might fix come handy if this gets annoying
                comp: [
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                    empty_pricecomp!(),
                ],
            }
        };
    }

    #[test]
    fn test_parse_pyth_price_wrong_size_slices() {
        assert!(parse_pyth_price(&[]).is_err());
        assert!(parse_pyth_price(vec![0u8; 1].as_slice()).is_err());
    }

    #[test]
    fn test_normal_values() -> SoliResult<()> {
        let price = Price {
	    expo: 5,
            agg: PriceInfo {
                price: 42,
                ..empty_priceinfo!()
            },
            ..empty_price!()
        };
        let price_vec = vec![price.clone()];

        // use the C repr to mock pyth's format
        let (_, bytes, _) = unsafe { price_vec.as_slice().align_to::<u8>() };

        assert_eq!(parse_pyth_price(bytes)?, price);
        Ok(())
    }
}
