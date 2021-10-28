pub mod pyth_extensions;

use std::{
    convert::{
        TryFrom,
        TryInto,
    },
    io::Read,
    mem,
};

use borsh::BorshSerialize;
use pyth_client::{
    AccountType,
    CorpAction,
    Ema,
    Price,
    PriceStatus,
    PriceType,
};
use solana_program::{
    clock::UnixTimestamp,
    program_error::ProgramError,
    pubkey::Pubkey,
};
use solitaire::{
    trace,
    ErrBox,
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

pub const PUBKEY_LEN: usize = 32;

/// Decides the format of following bytes
#[repr(u8)]
pub enum PayloadId {
    PriceAttestation = 1,
}

// On-chain data types

#[derive(Clone, Default, Debug, Eq, PartialEq)]
#[cfg_attr(feature = "wasm", derive(serde_derive::Serialize, serde_derive::Deserialize))]
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
    pub fn from_pyth_price_bytes(
        price_id: Pubkey,
        timestamp: UnixTimestamp,
        value: &[u8],
    ) -> Result<Self, SolitaireError> {
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
            timestamp,
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
    pub fn deserialize(mut bytes: impl Read) -> Result<Self, ErrBox> {
        use P2WCorpAction::*;
        use P2WPriceStatus::*;
        use P2WPriceType::*;

        let mut magic_vec = vec![0u8; P2W_MAGIC.len()];

        bytes.read_exact(magic_vec.as_mut_slice())?;

        if magic_vec.as_slice() != P2W_MAGIC {
            return Err(format!(
                "Invalid magic {:02X?}, expected {:02X?}",
                magic_vec, P2W_MAGIC,
            )
            .into());
        }

        let mut version_vec = vec![0u8; mem::size_of_val(&P2W_FORMAT_VERSION)];
        bytes.read_exact(version_vec.as_mut_slice())?;
        let version = u16::from_be_bytes(version_vec.as_slice().try_into()?);

        if version != P2W_FORMAT_VERSION {
            return Err(format!(
                "Unsupported format version {}, expected {}",
                version, P2W_FORMAT_VERSION
            )
            .into());
        }

        let mut payload_id_vec = vec![0u8; mem::size_of::<PayloadId>()];
        bytes.read_exact(payload_id_vec.as_mut_slice())?;

        if PayloadId::PriceAttestation as u8 != payload_id_vec[0] {
            return Err(format!(
                "Invalid Payload ID {}, expected {}",
                payload_id_vec[0],
                PayloadId::PriceAttestation as u8,
            )
            .into());
        }

        let mut product_id_vec = vec![0u8; PUBKEY_LEN];
        bytes.read_exact(product_id_vec.as_mut_slice())?;
        let product_id = Pubkey::new(product_id_vec.as_slice());

        let mut price_id_vec = vec![0u8; PUBKEY_LEN];
        bytes.read_exact(price_id_vec.as_mut_slice())?;
        let price_id = Pubkey::new(price_id_vec.as_slice());

        let mut price_type_vec = vec![0u8; mem::size_of::<P2WPriceType>()];
        bytes.read_exact(price_type_vec.as_mut_slice())?;
        let price_type = match price_type_vec[0] {
            a if a == Price as u8 => Price,
            a if a == P2WPriceType::Unknown as u8 => P2WPriceType::Unknown,
            other => {
                return Err(format!("Invalid price_type value {}", other).into());
            }
        };

        let mut price_vec = vec![0u8; mem::size_of::<i64>()];
	bytes.read_exact(price_vec.as_mut_slice())?;
	let price = i64::from_be_bytes(price_vec.as_slice().try_into()?);

        let mut expo_vec = vec![0u8; mem::size_of::<i32>()];
	bytes.read_exact(expo_vec.as_mut_slice())?;
	let expo = i32::from_be_bytes(expo_vec.as_slice().try_into()?);

	let twap = P2WEma::deserialize(&mut bytes)?;
	let twac = P2WEma::deserialize(&mut bytes)?;

	println!("twac OK");
        let mut confidence_interval_vec = vec![0u8; mem::size_of::<u64>()];
	bytes.read_exact(confidence_interval_vec.as_mut_slice())?;
	let confidence_interval = u64::from_be_bytes(confidence_interval_vec.as_slice().try_into()?);

        let mut status_vec = vec![0u8; mem::size_of::<P2WPriceType>()];
        bytes.read_exact(status_vec.as_mut_slice())?;
        let status = match status_vec[0] {
            a if a == P2WPriceStatus::Unknown as u8 => P2WPriceStatus::Unknown,
            a if a == Trading as u8 => Trading,
            a if a == Halted as u8 => Halted,
            a if a == Auction as u8 => Auction,
            other => {
                return Err(format!("Invalid status value {}", other).into());
            }
        };


        let mut corp_act_vec = vec![0u8; mem::size_of::<P2WPriceType>()];
        bytes.read_exact(corp_act_vec.as_mut_slice())?;
        let corp_act = match corp_act_vec[0] {
            a if a == NoCorpAct as u8 => NoCorpAct,
            other => {
                return Err(format!("Invalid corp_act value {}", other).into());
            }
        };

	let mut timestamp_vec = vec![0u8; mem::size_of::<UnixTimestamp>()];
			 bytes.read_exact(timestamp_vec.as_mut_slice())?;
	let timestamp = UnixTimestamp::from_be_bytes(timestamp_vec.as_slice().try_into()?);

	Ok( Self {
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
	})
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
    fn test_serialize_deserialize() -> Result<(), ErrBox> {
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
	let bytes = attestation.serialize();
        println!("Hex Bytes: {:02X?}", bytes);

	assert_eq!(PriceAttestation::deserialize(bytes.as_slice())?, attestation);
        Ok(())
    }
}
