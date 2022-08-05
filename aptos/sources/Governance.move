module Wormhole::Governance {
    use Wormhole::Deserialize;
    use Wormhole::VAA::{Self};
    use Wormhole::State::{updateGuardianSetIndex, storeGuardianSet, expireGuardianSet, getGuardianSet};
    use Wormhole::Structs::{Guardian, GuardianSet, createGuardian, createGuardianSet, getGuardianSetIndex};
    use 0x1::signer::{Self};
    //use AptosFramework::Table;
    use 0x1::vector::{Self};
    //use Std::String;
    //use sui::transfer;
    //use sui::vec_map::{Self, VecMap};

    const E_WRONG_GUARDIAN_LEN: u64 = 0x0;
    const E_REMAINING_BYTES: u64    = 0x1;
    const E_NO_GUARDIAN_SET: u64    = 0x2;

    struct GuardianUpdate has key{
        guardian_module:    vector<u8>,
        action:             u8,
        new_index:          u64,
        //guardians:          Table::Table<u8, Guardian>,
        guardians:          vector<Guardian>,
    }

    // public fun init_guardian_set(admin: &signer){
    //     //let x = Table::new<u8, Guardian>();
    //     let x = vector::empty<Guardian>();
    //     let addr = signer::address_of(admin);
    //     //assert!(!exists<GuardianSet>(addr), E_NO_GUARDIAN_SET);
    //     //move_to(admin, createGuardianSet(index, x));
    // }
    
    // Creates a new guardian set object with the given index. 
    public fun update_guardian_set(vaa: vector<u8>){
        // Verify VAA.
        let vaa = VAA::parse(vaa);

        //TODO: verify vaaa
        VAA::verifyVAA(&vaa, getGuardianSet());

        let payload = VAA::destroy(vaa);
        //let addr = signer::address_of(admin);
        //assert!(exists<GuardianSet>(addr), E_NO_GUARDIAN_SET);
        //let old = borrow_global_mut<GuardianSet>(addr);

        // Verify Governance Update.
        let update = parse(payload);

        verify(&update, getGuardianSet());

        let GuardianUpdate {
            guardian_module,
            action,
            new_index,  
            guardians,
        } = update;

        updateGuardianSetIndex(new_index);
        storeGuardianSet(createGuardianSet(new_index, guardians), new_index);
        expireGuardianSet(new_index-1);
    }

    public fun parse(bytes: vector<u8>): GuardianUpdate {
        //let guardians = Table::new<u8, Guardian>();
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
            //Table::add(&mut guardians, guardian_len-1, Guardian { key:key});
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
    
    public fun verify(update: &GuardianUpdate, previous: GuardianSet){
        let (guardian_module, action) = (update.guardian_module, update.action);
        assert!(vector::length(&guardian_module) == 32, 0);
        assert!(action == 0x02, 0); 
        assert!(update.new_index > getGuardianSetIndex(previous), 0);
    }
}