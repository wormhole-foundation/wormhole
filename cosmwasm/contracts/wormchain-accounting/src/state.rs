use accounting::state::transfer;
use cosmwasm_schema::cw_serde;
use cosmwasm_std::Binary;
use cw_storage_plus::Map;
use tinyvec::TinyVec;

use crate::msg::Observation;

pub const PENDING_TRANSFERS: Map<transfer::Key, TinyVec<[Data; 2]>> = Map::new("pending_transfers");
pub const CHAIN_REGISTRATIONS: Map<u16, Binary> = Map::new("chain_registrations");
pub const DIGESTS: Map<(u16, Vec<u8>, u64), Binary> = Map::new("digests");

#[cw_serde]
pub struct PendingTransfer {
    pub key: transfer::Key,
    pub data: Vec<Data>,
}

#[cw_serde]
#[derive(Default)]
pub struct Data {
    observation: Observation,
    guardian_set_index: u32,
    signatures: u128,
}

impl Data {
    pub const fn new(observation: Observation, guardian_set_index: u32) -> Self {
        Self {
            observation,
            guardian_set_index,
            signatures: 0,
        }
    }

    pub fn observation(&self) -> &Observation {
        &self.observation
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
