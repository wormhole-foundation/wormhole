use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::{
        Seeded,
        SingleOwned,
    },
    AccountOwner,
    AccountState,
    Data,
    Owned,
};

pub type Sequence<'b> = Data<'b, SequenceTracker, { AccountState::MaybeInitialized }>;

/// Account discriminator for SequenceTracker — prevents type confusion attacks.
pub const SEQUENCE_DISCRIMINATOR: &[u8] = b"seqt";

#[derive(Default, Clone, Copy)]
pub struct SequenceTracker {
    pub sequence: u64,
}

impl BorshSerialize for SequenceTracker {
    fn serialize<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write_all(SEQUENCE_DISCRIMINATOR)?;
        BorshSerialize::serialize(&self.sequence, writer)
    }
}

impl BorshDeserialize for SequenceTracker {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        use std::io::{Error, ErrorKind::InvalidData};
        if buf.len() < SEQUENCE_DISCRIMINATOR.len() {
            return Err(Error::new(InvalidData, "Not enough bytes for SequenceTracker discriminator"));
        }
        let magic = &buf[..SEQUENCE_DISCRIMINATOR.len()];
        if magic != SEQUENCE_DISCRIMINATOR {
            return Err(Error::new(InvalidData, format!("SequenceTracker discriminator mismatch. Expected {:?} but got {:?}", SEQUENCE_DISCRIMINATOR, magic)));
        }
        *buf = &buf[SEQUENCE_DISCRIMINATOR.len()..];
        Ok(SequenceTracker {
            sequence: BorshDeserialize::deserialize(buf)?,
        })
    }
}

pub struct SequenceDerivationData<'a> {
    pub emitter_key: &'a Pubkey,
}

impl<'b> Seeded<&SequenceDerivationData<'b>> for Sequence<'b> {
    fn seeds(data: &SequenceDerivationData) -> Vec<Vec<u8>> {
        vec![
            "Sequence".as_bytes().to_vec(),
            data.emitter_key.to_bytes().to_vec(),
        ]
    }
}

impl Owned for SequenceTracker {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

impl SingleOwned for SequenceTracker {
}
