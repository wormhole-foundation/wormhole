module Wormhole::Governance {
    use Wormhole::Deserialize;
    use Wormhole::VAA::{Self};
    use Wormhole::State::{updateGuardianSetIndex, storeGuardianSet, expireGuardianSet, getCurrentGuardianSet, getCurrentGuardianSetIndex};
    use Wormhole::Structs::{Guardian, GuardianSet, createGuardian, createGuardianSet, getGuardianSetIndex};
    use 0x1::vector::{Self};
    use 0x1::string::{Self, String};

    const E_WRONG_GUARDIAN_LEN: u64 = 0x0;
    const E_REMAINING_BYTES: u64    = 0x1;
    const E_NO_GUARDIAN_SET: u64    = 0x2;

    struct GuardianUpdate has key{
        guardian_module:    vector<u8>, 
        action:             u8,
        new_index:          u64,
        guardians:          vector<Guardian>,
    }
    
    public entry fun update_guardian_set(vaa: vector<u8>): (bool, String){
        let (vaa, valid, reason) = VAA::parseAndVerifyVAA(vaa);

        let payload = VAA::destroy(vaa);
        
        if (valid==false){
            return (false, reason)
        };

        // Verify Governance Update.
        let update = parse(payload);

        verify(&update, getCurrentGuardianSet());

        let GuardianUpdate {
            guardian_module,
            action, //action
            new_index,  
            guardians,
        } = update;

        updateGuardianSetIndex(new_index);
        storeGuardianSet(createGuardianSet(new_index, guardians), new_index);
        expireGuardianSet(new_index-1);
        return (true, string::utf8(b""))
    }

    public entry fun parse(bytes: vector<u8>): GuardianUpdate {
        let guardians = vector::empty<Guardian>();
        let (guardian_module, bytes) = Deserialize::deserialize_vector(bytes, 32);
        //TODO: missing chainID?
        let (action, bytes) = Deserialize::deserialize_u8(bytes);
        let (new_index, bytes) = Deserialize::deserialize_u64(bytes);
        let (guardian_len, bytes) = Deserialize::deserialize_u8(bytes);

        assert!(guardian_len < 19, E_WRONG_GUARDIAN_LEN);

        while ({
            spec {
                invariant guardian_len >= 0;
                invariant guardian_len < 19;
            };
            guardian_len > 0
        }) { 
            let (key, bytes) = Deserialize::deserialize_vector(bytes, 20);
            vector::push_back(&mut guardians, createGuardian(key));
            guardian_len = guardian_len - 1;
        };

        assert!(vector::length(&mut bytes) == 0, E_REMAINING_BYTES);

        GuardianUpdate {
            guardian_module:    guardian_module,
            action:             action,
            new_index:          new_index,
            guardians:          guardians,
        }
    }
    
    public entry fun verify(update: &GuardianUpdate, previous: GuardianSet){
        let (guardian_module, action) = (update.guardian_module, update.action);
        assert!(vector::length(&guardian_module) == 32, 0);
        assert!(action == 0x02, 0); 
        assert!(update.new_index > getGuardianSetIndex(previous), 0);
    }
}