module Wormhole::Structs{
    use Wormhole::u32::{Self, U32};
    use 0x1::secp256k1::{Self};

    friend Wormhole::State;
    friend Wormhole::VAA;
    friend Wormhole::Wormhole;
    use Wormhole::guardian_pubkey::{Self};

    //friend Wormhole::Governance;
    //friend Wormhole::Wormhole;

    struct Signature has key, store, copy, drop {
        sig: secp256k1::ECDSASignature,
        recovery_id: u8,
        guardianIndex: U32,
    }

    struct Guardian has key, store, drop, copy {
        address: guardian_pubkey::Address
    }

    struct GuardianSet has key, store, copy, drop {
        index:     U32,
        guardians: vector<Guardian>,
        expirationTime: U32,
    }

    public fun createGuardian(address: vector<u8>): Guardian {
        Guardian{
            address: guardian_pubkey::from_bytes(address)
        }
    }

    public fun createGuardianSet(index: U32, guardians: vector<Guardian>): GuardianSet{
        GuardianSet{
            index: index,
            guardians: guardians,
            expirationTime: u32::from_u64(0),
        }
    }

    public(friend) fun expireGuardianSet(_guardianSet: &mut GuardianSet){
        // TODO - right now U32 addition not supported
        //guardianSet.expirationTime = timestamp::now_seconds() + 86400;
    }

    public fun unpackSignature(s: &Signature): (secp256k1::ECDSASignature, u8, U32) {
        (s.sig, s.recovery_id, s.guardianIndex)
    }

    public fun createSignature(
        sig: secp256k1::ECDSASignature,
        recovery_id: u8,
        guardianIndex: U32
    ): Signature {
        Signature{ sig, recovery_id, guardianIndex }
    }

    public fun getAddress(guardian: Guardian): guardian_pubkey::Address {
        guardian.address
    }

    public fun getGuardianSetIndex(guardianSet: GuardianSet): U32 {
        guardianSet.index
    }

    public fun getGuardians(guardianSet: GuardianSet): vector<Guardian> {
        guardianSet.guardians
    }

    public fun getGuardianSetExpiry(guardianSet: GuardianSet): U32{
        guardianSet.expirationTime
    }

}
