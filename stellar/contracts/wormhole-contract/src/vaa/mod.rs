mod signature;

pub use signature::Signature;

use crate::{governance::guardian_set, utils::BytesReader};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env, Vec, contracttype};
use wormhole_interface::Error;

#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct VAA {
    // Header
    pub version: u32,
    pub guardian_set_index: u32,
    pub signatures: Vec<Signature>,

    // Body
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain: u32,
    pub emitter_address: BytesN<32>,
    pub sequence: u64,
    pub consistency_level: u32,
    pub payload: Bytes,
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for VAA {
    type Error = Error;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, vaa_bytes) = value;

        if vaa_bytes.len() < 6 {
            return Err(Error::InvalidVAAFormat);
        }

        let mut reader = BytesReader::new(vaa_bytes);

        let version = u32::from(reader.read_u8()?);
        let guardian_set_index = reader.read_u32_be()?;
        let num_signatures = u32::from(reader.read_u8()?);

        let mut signatures = Vec::new(env);
        for _ in 0..num_signatures {
            let sig = Signature::parse(env, &mut reader)?;
            signatures.push_back(sig);
        }

        let timestamp = reader.read_u32_be()?;
        let nonce = reader.read_u32_be()?;
        let emitter_chain = u32::from(reader.read_u16_be()?);
        let emitter_address = reader.read_bytes_n::<32>(env)?;
        let sequence = reader.read_u64_be()?;
        let consistency_level = u32::from(reader.read_u8()?);
        let payload = reader.remaining_bytes();

        Ok(VAA {
            version,
            guardian_set_index,
            signatures,
            timestamp,
            nonce,
            emitter_chain,
            emitter_address,
            sequence,
            consistency_level,
            payload,
        })
    }
}

impl VAA {
    /// Serialize the VAA body for hashing
    #[allow(clippy::arithmetic_side_effects, clippy::cast_possible_truncation)]
    pub(crate) fn serialize_body(&self, env: &Env) -> Bytes {
        let mut bytes = Bytes::new(env);

        for i in (0..4).rev() {
            bytes.push_back(((self.timestamp >> (i * 8)) & 0xFF) as u8);
        }

        for i in (0..4).rev() {
            bytes.push_back(((self.nonce >> (i * 8)) & 0xFF) as u8);
        }

        bytes.push_back((self.emitter_chain >> 8) as u8);
        bytes.push_back((self.emitter_chain & 0xFF) as u8);

        for byte in self.emitter_address.to_array().iter() {
            bytes.push_back(*byte);
        }

        for i in (0..8).rev() {
            bytes.push_back(((self.sequence >> (i * 8)) & 0xFF) as u8);
        }

        bytes.push_back(self.consistency_level as u8);

        bytes.append(&self.payload);

        bytes
    }

    /// Get the body bytes for hashing (everything after signatures)
    pub(crate) fn get_body_bytes(vaa_bytes: &Bytes) -> Result<Bytes, Error> {
        if vaa_bytes.len() < 6 {
            return Err(Error::InvalidVAAFormat);
        }

        let num_sigs = u32::from(vaa_bytes.get(5).ok_or(Error::InvalidVAAFormat)?);
        let body_offset = 6u32.saturating_add(66u32.saturating_mul(num_sigs));

        if body_offset >= vaa_bytes.len() {
            return Err(Error::InvalidVAAFormat);
        }

        Ok(vaa_bytes.slice(body_offset..))
    }

    /// Verify this VAA's signatures against stored guardian sets
    pub(crate) fn verify(&self, env: &Env) -> Result<bool, Error> {
        let body_bytes = self.serialize_body(env);

        let guardian_set_info = guardian_set::get(env, self.guardian_set_index)?;

        if let Some(expiry) = guardian_set::get_expiry(env, self.guardian_set_index)
            && env.ledger().timestamp() > expiry
        {
            return Err(Error::GuardianSetExpired);
        }

        self.verify_signatures(env, &body_bytes, &guardian_set_info.keys)
    }

    /// Calculate the required quorum for a given number of guardians.
    /// Formula: (num_guardians * 2 / 3) + 1
    fn calculate_quorum(num_guardians: u32) -> u32 {
        num_guardians
            .saturating_mul(2)
            .saturating_div(3)
            .saturating_add(1)
    }

    /// Verify signatures against a specific guardian set
    fn verify_signatures(
        &self,
        env: &Env,
        body_bytes: &Bytes,
        guardian_keys: &Vec<BytesN<20>>,
    ) -> Result<bool, Error> {
        let guardian_count = guardian_keys.len();
        if guardian_count == 0 {
            return Err(Error::InvalidGuardianSetIndex);
        }

        let required_sigs = VAA::calculate_quorum(guardian_count);
        if self.signatures.len() < required_sigs {
            return Err(Error::InsufficientSignatures);
        }

        let body_hash_bytes: Bytes = crate::utils::keccak256_hash(env, body_bytes).into();
        let double_hash = env.crypto().keccak256(&body_hash_bytes);

        let mut last_guardian_index = None;

        for signature in self.signatures.iter() {
            if let Some(last_idx) = last_guardian_index
                && signature.guardian_index <= last_idx
            {
                return Err(Error::SignaturesNotAscending);
            }
            last_guardian_index = Some(signature.guardian_index);

            if signature.guardian_index >= guardian_count {
                return Err(Error::GuardianIndexOutOfBounds);
            }

            let guardian_key = guardian_keys
                .get(signature.guardian_index)
                .ok_or(Error::GuardianIndexOutOfBounds)?;

            if !signature.verify(env, &double_hash, &guardian_key)? {
                return Err(Error::InvalidSignature);
            }
        }

        Ok(true)
    }
}

pub(crate) fn verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<bool, Error> {
    let vaa = VAA::try_from((&env, &vaa_bytes))?;
    vaa.verify(&env)
}

/// Parse a VAA and convert to the interface type
pub(crate) fn parse_vaa(env: &Env, vaa_bytes: &Bytes) -> Result<wormhole_interface::VAA, Error> {
    let internal_vaa = VAA::try_from((env, vaa_bytes))?;

    // Convert internal signatures to interface signatures
    let mut interface_signatures = Vec::new(env);
    for sig in internal_vaa.signatures.iter() {
        interface_signatures.push_back(wormhole_interface::Signature {
            guardian_index: sig.guardian_index,
            r: sig.r.clone(),
            s: sig.s.clone(),
            v: sig.v,
        });
    }

    Ok(wormhole_interface::VAA {
        version: internal_vaa.version,
        guardian_set_index: internal_vaa.guardian_set_index,
        signatures: interface_signatures,
        timestamp: internal_vaa.timestamp,
        nonce: internal_vaa.nonce,
        emitter_chain: internal_vaa.emitter_chain,
        emitter_address: internal_vaa.emitter_address,
        sequence: internal_vaa.sequence,
        consistency_level: internal_vaa.consistency_level,
        payload: internal_vaa.payload,
    })
}
