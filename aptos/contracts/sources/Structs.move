module Wormhole::Structs{
    use 0x1::timestamp::{Self};
    use 0x1::vector::{Self};
    use Wormhole::Uints::{U16, U32, zero_u32};

    friend Wormhole::State;
    friend Wormhole::VAA;
    friend Wormhole::Wormhole;

    //friend Wormhole::Governance;
    //friend Wormhole::Wormhole;

    struct Signature has key, store, copy, drop{
        signature: vector<u8>,
        guardianIndex: U32,
	}

    struct Guardian has key, store, drop, copy {
        address: vector<u8>,
    }

    struct GuardianSet has key, store, copy, drop {
        index:     U32,
        guardians: vector<Guardian>,
        expirationTime: U32,
    }

    public fun createGuardian(address: vector<u8>): Guardian {
        Guardian{
            address: address
        }
    }

    public fun createGuardianSet(index: U32, guardians: vector<Guardian>): GuardianSet{
        GuardianSet{
            index: index,
            guardians: guardians,
            expirationTime: zero_u32(), // represents 0 as a U32
        }
    }

    public(friend) fun expireGuardianSet(guardianSet: &mut GuardianSet){
        // TODO - right now U32 addition not supported
        //guardianSet.expirationTime = timestamp::now_seconds() + 86400;
    }

    public fun unpackSignature(s: &Signature): (vector<u8>, U32){
        (s.signature,  s.guardianIndex)
    }

    public fun createSignature(s: vector<u8>, guardianIndex: U32): Signature{
        Signature{
            signature:      s,
            guardianIndex:  guardianIndex,
        }
    }

    public fun getAddress(guardian: Guardian): vector<u8> {
        guardian.address
    }

    public fun getGuardianSetIndex(guardianSet: GuardianSet): U32{
        guardianSet.index
    }

    public fun getGuardians(guardianSet: GuardianSet): vector<Guardian> {
        guardianSet.guardians
    }

    public fun getGuardianSetExpiry(guardianSet: GuardianSet): U32{
        guardianSet.expirationTime
    }

}
