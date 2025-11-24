use crate::{utils::BytesReader, utils::pubkey_to_eth_address};
use soroban_sdk::{contracttype, BytesN, Env};
use wormhole_interface::Error;

#[contracttype]
#[derive(Clone, Debug, PartialEq)]
pub struct Signature {
    pub guardian_index: u32,
    pub r: BytesN<32>,
    pub s: BytesN<32>,
    pub v: u32,
}

impl Signature {
    pub fn parse(env: &Env, reader: &mut BytesReader) -> Result<Self, Error> {
        let guardian_index = u32::from(reader.read_u8()?);
        let r = reader.read_bytes_n::<32>(env)?;
        let s = reader.read_bytes_n::<32>(env)?;
        let v = u32::from(reader.read_u8()?);

        Ok(Signature {
            guardian_index,
            r,
            s,
            v,
        })
    }

    pub fn verify(
        &self,
        env: &Env,
        message_hash: &soroban_sdk::crypto::Hash<32>,
        expected_address: &BytesN<20>,
    ) -> Result<bool, Error> {
        let mut sig_bytes = [0u8; 64];
        let r_array = self.r.to_array();
        let s_array = self.s.to_array();
        sig_bytes[..32].copy_from_slice(&r_array);
        sig_bytes[32..].copy_from_slice(&s_array);
        let sig = BytesN::<64>::from_array(env, &sig_bytes);

        let recovered_pubkey = env
            .crypto()
            .secp256k1_recover(message_hash, &sig, self.v);

        let eth_address = pubkey_to_eth_address(env, &recovered_pubkey);

        Ok(&eth_address == expected_address)
    }
}
