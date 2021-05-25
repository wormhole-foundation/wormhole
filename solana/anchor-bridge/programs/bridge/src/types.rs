use borsh::{BorshDeserialize, BorshSerialize};

#[derive(Default, Clone, Copy, BorshSerialize, BorshDeserialize)]
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
