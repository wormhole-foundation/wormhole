use soroban_sdk::{Address, BytesN, contracttype};

#[derive(Clone)]
#[contracttype]
pub(crate) enum StorageKey {
    /// Current active guardian set index
    CurrentGuardianSetIndex,
    /// Whether the contract is initialized
    Initialized,
    /// Admin address for contract upgrades (set to contract itself)
    Admin,
    /// Guardian set information by index
    GuardianSet(u32),
    /// Guardian set expiry timestamp by index
    GuardianSetExpiry(u32),
    /// Consumed governance VAA hashes for replay protection
    ConsumedGovernanceVAA(BytesN<32>),
    /// Current message fee in stroops (smallest XLM unit)
    MessageFee,
    /// Last fee transfer timestamp (for audit)
    LastFeeTransfer,
    /// Sequence number for an emitter address
    EmitterSequence(Address),
    /// Posted message by sequence, (emitter, sequence)
    PostedMessage(Address, u64),
}
