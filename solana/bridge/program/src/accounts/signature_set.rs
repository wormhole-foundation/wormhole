//! PostedMessage

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solitaire::{
    AccountOwner,
    AccountState,
    Data,
    Owned,
};
use std::io::{
    Error,
    ErrorKind::InvalidData,
    Write,
};

pub type SignatureSet<'b, const State: AccountState> = Data<'b, SignatureSetData, { State }>;

/// Account discriminator for SignatureSetData — prevents type confusion attacks.
pub const SIGNATURE_SET_DISCRIMINATOR: &[u8] = b"sigs";

#[derive(Default)]
pub struct SignatureSetData {
    /// Signatures of validators
    pub signatures: Vec<bool>,

    /// Hash of the data
    pub hash: [u8; 32],

    /// Index of the guardian set
    pub guardian_set_index: u32,
}

impl BorshSerialize for SignatureSetData {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write_all(SIGNATURE_SET_DISCRIMINATOR)?;
        BorshSerialize::serialize(&self.signatures, writer)?;
        BorshSerialize::serialize(&self.hash, writer)?;
        BorshSerialize::serialize(&self.guardian_set_index, writer)
    }
}

impl BorshDeserialize for SignatureSetData {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        if buf.len() < SIGNATURE_SET_DISCRIMINATOR.len() {
            return Err(Error::new(InvalidData, "Not enough bytes for SignatureSetData discriminator"));
        }
        let magic = &buf[..SIGNATURE_SET_DISCRIMINATOR.len()];
        if magic != SIGNATURE_SET_DISCRIMINATOR {
            return Err(Error::new(
                InvalidData,
                format!("SignatureSetData discriminator mismatch. Expected {:?} but got {:?}", SIGNATURE_SET_DISCRIMINATOR, magic),
            ));
        }
        *buf = &buf[SIGNATURE_SET_DISCRIMINATOR.len()..];
        Ok(SignatureSetData {
            signatures: BorshDeserialize::deserialize(buf)?,
            hash: BorshDeserialize::deserialize(buf)?,
            guardian_set_index: BorshDeserialize::deserialize(buf)?,
        })
    }
}

impl Owned for SignatureSetData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}
