pub mod pyth_extensions;

use std::mem;

use borsh::BorshSerialize;
use pyth_client::{
    AccountType,
    CorpAction,
    Ema,
    Price,
    PriceStatus,
    PriceType,
};
use solana_program::{clock::UnixTimestamp, program_error::ProgramError, pubkey::Pubkey};
use solitaire::{
    trace,
    Result as SoliResult,
    SolitaireError,
};

use self::pyth_extensions::{
    P2WCorpAction,
    P2WEma,
    P2WPriceStatus,
    P2WPriceType,
};

// Constants and values common to every p2w custom-serialized message

/// Precedes every message implementing the p2w serialization format
pub const P2W_MAGIC: &'static [u8] = b"P2WH";

/// Format version used and understood by this codebase
pub const P2W_FORMAT_VERSION: u16 = 1;

/// Decides the format of following bytes
#[repr(u8)]
pub enum PayloadId {
    PriceAttestation = 1,
}

// On-chain data types

#[derive(Clone, Default, Debug, Eq, PartialEq)]
pub struct PriceAttestation {
    pub product_id: Pubkey,
    pub price_id: Pubkey,
    pub price_type: P2WPriceType,
    pub price: i64,
    pub expo: i32,
    pub twap: P2WEma,
    pub twac: P2WEma,
    pub confidence_interval: u64,
    pub status: P2WPriceStatus,
    pub corp_act: P2WCorpAction,
    pub timestamp: UnixTimestamp,
}

impl PriceAttestation {
    pub fn from_pyth_price_bytes(price_id: Pubkey, timestamp: UnixTimestamp, value: &[u8]) -> Result<Self, SolitaireError> {
        let price = parse_pyth_price(value)?;

        Ok(PriceAttestation {
            product_id: Pubkey::new(&price.prod.val[..]),
            price_id,
            price_type: (&price.ptype).into(),
            price: price.agg.price,
            twap: (&price.twap).into(),
            twac: (&price.twac).into(),
            expo: price.expo,
            confidence_interval: price.agg.conf,
            status: (&price.agg.status).into(),
            corp_act: (&price.agg.corp_act).into(),
	    timestamp: timestamp,
        })
    }

    /// Serialize this attestation according to the Pyth-over-wormhole serialization format
    pub fn serialize(&self) -> Vec<u8> {
        // A nifty trick to get us yelled at if we forget to serialize a field
	#[deny(warnings)]
        let PriceAttestation {
            product_id,
            price_id,
            price_type,
            price,
            expo,
            twap,
            twac,
            confidence_interval,
            status,
            corp_act,
	    timestamp
        } = self;

        // magic
        let mut buf = P2W_MAGIC.to_vec();

        // version
        buf.extend_from_slice(&P2W_FORMAT_VERSION.to_be_bytes()[..]);

        // payload_id
        buf.push(PayloadId::PriceAttestation as u8);

        // product_id
        buf.extend_from_slice(&product_id.to_bytes()[..]);

        // price_id
        buf.extend_from_slice(&price_id.to_bytes()[..]);

        // price_type
        buf.push(price_type.clone() as u8);

        // price
        buf.extend_from_slice(&price.to_be_bytes()[..]);

        // exponent
        buf.extend_from_slice(&expo.to_be_bytes()[..]);

        // twap
        buf.append(&mut twap.serialize());

        // twac
        buf.append(&mut twac.serialize());

	// confidence_interval
	buf.extend_from_slice(&confidence_interval.to_be_bytes()[..]);

	// status
	buf.push(status.clone() as u8);

	// corp_act
	buf.push(corp_act.clone() as u8);

	// timestamp
	buf.extend_from_slice(&timestamp.to_be_bytes()[..]);

        buf
    }
}

/// Deserializes Price from raw bytes, sanity-check.
fn parse_pyth_price(price_data: &[u8]) -> SoliResult<&Price> {
    if price_data.len() != mem::size_of::<Price>() {
        trace!(&format!(
            "parse_pyth_price: buffer length mismatch ({} expected, got {})",
            mem::size_of::<Price>(),
            price_data.len()
        ));
        return Err(ProgramError::InvalidAccountData.into());
    }
    let price_account = pyth_client::cast::<Price>(price_data);

    if price_account.atype != AccountType::Price as u32 {
        trace!(&format!(
            "parse_pyth_price: AccountType mismatch ({} expected, got {})",
            mem::size_of::<Price>(),
            price_data.len()
        ));
        return Err(ProgramError::InvalidAccountData.into());
    }

    Ok(price_account)
}

#[cfg(test)]
mod tests {
    use super::*;
    use pyth_client::{
        AccKey,
        AccountType,
        PriceComp,
        PriceInfo,
    };

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

    macro_rules! empty_ema {
        () => {
            (&P2WEma::default()).into()
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
                num_qt: 0,
                last_slot: 0,
                valid_slot: 0,
                drv1: 0,
                drv2: 0,
                drv3: 0,
                twap: empty_ema!(),
                twac: empty_ema!(),
                prod: empty_acckey!(),
                next: empty_acckey!(),
                prev_slot: 0,  // valid slot of previous update
                prev_price: 0, // aggregate price of previous update
                prev_conf: 0,  // confidence interval of previous update
                agg: empty_priceinfo!(),
                // A nice macro might come in handy if this gets annoying
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
        let price_vec = vec![price];

        // use the C repr to mock pyth's format
        let (_, bytes, _) = unsafe { price_vec.as_slice().align_to::<u8>() };

        parse_pyth_price(bytes)?;
        Ok(())
    }

    #[test]
    fn test_serialize() -> SoliResult<()> {
        let product_id_bytes = [21u8; 32];
        let price_id_bytes = [222u8; 32];
        println!("Hex product_id: {:02X?}", &product_id_bytes);
        println!("Hex price_id: {:02X?}", &price_id_bytes);
        let attestation: PriceAttestation = PriceAttestation {
            product_id: Pubkey::new_from_array(product_id_bytes),
            price_id: Pubkey::new_from_array(price_id_bytes),
            price: (0xdeadbeefdeadbabe as u64) as i64,
	    price_type: P2WPriceType::Price,
            twap: P2WEma {
                val: -42,
                numer: 15,
                denom: 37,
            },
            twac: P2WEma {
                val: 42,
                numer: 1111,
                denom: 2222,
            },
            expo: -3,
	    status: P2WPriceStatus::Trading,
            confidence_interval: 101,
	    corp_act: P2WCorpAction::NoCorpAct,
	    timestamp: 123456789i64,
        };

        println!("Regular: {:#?}", &attestation);
        println!("Hex: {:#02X?}", &attestation);
        println!("Hex Bytes: {:02X?}", attestation.serialize());
        Ok(())
    }
}
