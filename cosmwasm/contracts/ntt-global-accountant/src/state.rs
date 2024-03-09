use accountant::state::{transfer, TokenAddress};
use cosmwasm_schema::cw_serde;
use cosmwasm_std::Binary;
use cw_storage_plus::Map;
use tinyvec::TinyVec;

pub const PENDING_TRANSFERS: Map<transfer::Key, TinyVec<[Data; 2]>> = Map::new("pending_transfers");
pub const RELAYER_CHAIN_REGISTRATIONS: Map<u16, Binary> = Map::new("relayer_chain_registrations");
pub const TRANSCEIVER_TO_HUB: Map<(u16, TokenAddress), (u16, TokenAddress)> =
    Map::new("transceiver_to_hub");
pub const TRANSCEIVER_PEER: Map<(u16, TokenAddress, u16), TokenAddress> =
    Map::new("transceiver_peers");
pub const DIGESTS: Map<(u16, Vec<u8>, u64), Binary> = Map::new("digests");

#[cw_serde]
pub struct TransceiverHub {
    pub key: (u16, TokenAddress),
    pub data: (u16, TokenAddress),
}

#[cw_serde]
pub struct TransceiverPeer {
    pub key: (u16, TokenAddress, u16),
    pub data: TokenAddress,
}

#[cw_serde]
pub struct PendingTransfer {
    pub key: transfer::Key,
    pub data: Vec<Data>,
}

#[cw_serde]
#[derive(Default)]
pub struct Data {
    digest: Binary,
    tx_hash: Binary,
    signatures: u128,
    guardian_set_index: u32,
    emitter_chain: u16,
}

impl Data {
    pub const fn new(
        digest: Binary,
        tx_hash: Binary,
        emitter_chain: u16,
        guardian_set_index: u32,
    ) -> Self {
        Self {
            digest,
            tx_hash,
            signatures: 0,
            guardian_set_index,
            emitter_chain,
        }
    }

    pub fn digest(&self) -> &Binary {
        &self.digest
    }

    pub fn tx_hash(&self) -> &Binary {
        &self.tx_hash
    }

    pub fn emitter_chain(&self) -> u16 {
        self.emitter_chain
    }

    pub fn guardian_set_index(&self) -> u32 {
        self.guardian_set_index
    }

    pub fn signatures(&self) -> u128 {
        self.signatures
    }

    /// Returns the number of signatures for this `Data`.
    pub fn num_signatures(&self) -> u32 {
        self.signatures.count_ones()
    }

    /// Returns true if there is a signature associated with `index` in this `Data`.
    pub fn has_signature(&self, index: u8) -> bool {
        assert!(index < 128);
        self.signatures & (1u128 << index) != 0
    }

    /// Adds `sig` to the list of signatures for this `Data`.
    pub fn add_signature(&mut self, index: u8) {
        assert!(index < 128);
        self.signatures |= 1u128 << index;
    }
}

#[cfg(test)]
mod test {
    use super::*;

    use std::panic::catch_unwind;

    #[test]
    fn add_signatures() {
        let mut data = Data::default();
        for i in 0..128 {
            assert!(!data.has_signature(i));
            data.add_signature(i);
            assert!(data.has_signature(i));
            assert_eq!(u32::from(i + 1), data.num_signatures());
        }
    }

    #[test]
    fn add_signatures_rev() {
        let mut data = Data::default();
        for i in (0..128).rev() {
            assert!(!data.has_signature(i));
            data.add_signature(i);
            assert!(data.has_signature(i));
            assert_eq!(u32::from(128 - i), data.num_signatures());
        }
    }

    #[test]
    fn has_out_of_bounds_signature() {
        for i in 128..=u8::MAX {
            catch_unwind(|| Data::default().has_signature(i))
                .expect_err("successfully checked for out-of-bounds signature");
        }
    }

    #[test]
    fn add_out_of_bounds_signature() {
        for i in 128..=u8::MAX {
            catch_unwind(|| Data::default().add_signature(i))
                .expect_err("successfully added out-of-bounds signature");
        }
    }
}
