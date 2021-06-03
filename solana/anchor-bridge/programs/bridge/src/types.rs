use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::pubkey::Pubkey;

#[derive(BorshSerialize, BorshDeserialize, Clone, Copy, Default, PartialEq)]
pub struct Index(u8);

impl Index {
    pub fn new(n: u8) -> Self {
        Index(n)
    }

    pub fn bump(mut self) -> Self {
        self.0 += 1;
        self
    }
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct GuardianSetData {
    /// Version number of this guardian set.
    pub index: Index,

    /// public key hashes of the guardian set
    pub keys: Vec<u8>,

    /// creation time
    pub creation_time: u32,

    /// expiration time when VAAs issued by this set are no longer valid
    pub expiration_time: u32,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct BridgeConfig {
    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub guardian_set_expiration_time: u32,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct BridgeData {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub guardian_set_index: Index,

    /// Bridge configuration, which is set once upon initialization.
    pub config: BridgeConfig,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct Bridge {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub guardian_set_index: Index,

    /// Bridge configuration, which is set once upon initialization.
    pub config: BridgeConfig,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct SignatureSet {
    /// Signatures of validators
    pub signatures: Vec<[u8; 32]>,

    /// Hash of the data
    pub hash: [u8; 32],

    /// Index of the guardian set
    pub guardian_set_index: Index,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct GuardianSet {
    /// Index of this guardian set.
    pub index: Index,

    /// Public key hashes of the guardian set
    pub keys: Vec<[u8; 20]>,

    /// Creation time
    pub creation_time: u32,

    /// Expiration time when VAAs issued by this set are no longer valid
    pub expiration_time: u32,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct PostedMessage {
    /// Header of the posted VAA
    pub vaa_version: u8,

    /// Time the vaa was submitted
    pub vaa_time: u32,

    /// Account where signatures are stored
    pub vaa_signature_account: Pubkey,

    /// Time the posted message was created
    pub submission_time: u32,

    /// Unique nonce for this message
    pub nonce: u32,

    /// Emitter of the message
    pub emitter_chain: Chain,

    /// Emitter of the message
    pub emitter_address: [u8; 32],

    /// Message payload
    pub payload: Vec<u8>,
}

#[repr(u8)]
#[derive(BorshSerialize, BorshDeserialize)]
pub enum Chain {
    Unknown,
    Solana = 1u8,
}

impl Default for Chain {
    fn default() -> Self {
        Chain::Unknown
    }
}
