pub mod pyth_extensions;

use std::{
    convert::TryInto,
    io::Read,
    mem,
};

use solana_program::{
    clock::UnixTimestamp,
    pubkey::Pubkey,
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

#[derive(
    Clone, Default, Debug, Eq, PartialEq, serde_derive::Serialize, serde_derive::Deserialize,
)]
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
    pub fn deserialize(mut bytes: impl Read) -> Result<Self, Box<dyn std::error::Error>> {
        use P2WCorpAction::*;
        use P2WPriceStatus::*;
        use P2WPriceType::*;

        println!("Using {} bytes for magic", P2W_MAGIC.len());
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
        let confidence_interval =
            u64::from_be_bytes(confidence_interval_vec.as_slice().try_into()?);

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

        Ok(Self {
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
        })
    }
}
