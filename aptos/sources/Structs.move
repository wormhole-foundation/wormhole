module Wormhole::Structs{

    struct Signature {
        r: vector<u8>, 
        s: vector<u8>, 
        v: u8, 
        guardianIndex: u8, 
	}

    struct Guardian has key, store, drop, copy{
        key:       vector<u8>,
    }

    struct GuardianSet has key, copy, drop {
        index:     u64, 
        guardians: vector<Guardian>,
        //expirationTime: u64, //u32
    }

    public fun createGuardian(key: vector<u8>): Guardian{
        Guardian{
            key: key
        }
    }
    
    public fun createGuardianSet(index: u64, guardians: vector<Guardian>): GuardianSet{
        GuardianSet{
            index: index, 
            guardians: guardians,
        }
    }

    public fun getKey(guardian: Guardian): vector<u8>{
        guardian.key
    }
    
    public fun getGuardianSetIndex(guardianSet: GuardianSet): u64{
        guardianSet.index
    }

    public fun getGuardians(guardianSet: GuardianSet): vector<Guardian>{
        guardianSet.guardians
    }

} 