module Wormhole::Governance {
    use Wormhole::Deserialize;
    use Wormhole::VAA::{Self};
    use AptosFramework::Signer;
    //use AptosFramework::Table;
    use Std::Vector;
    
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
    
    struct Guardian has key, store, drop{
        key:       vector<u8>,
    }

    struct GuardianSet has key {
        index:     u64, 
        //guardians: Table::Table<u8, Guardian>,
        guardians: vector<Guardian>,
    }

    public fun init_guardian_set(admin: &signer){
        //let x = Table::new<u8, Guardian>();
        let x = Vector::empty<Guardian>();
        let addr = Signer::address_of(admin);
        assert!(!exists<GuardianSet>(addr), E_NO_GUARDIAN_SET);
        move_to(admin, GuardianSet {
             index:     0,
             guardians: x,
        });
    }
    
    // Creates a new guardian set object with the given index. 
    public fun update_guardian_set(admin: &signer, vaa: vector<u8>) acquires GuardianSet{
        // Verify VAA.
        let vaa = VAA::parse(vaa);

        //TODO: verify vaaa
        //VAA::verify(&vaa, &old);
        let payload = VAA::destroy(vaa);

        let addr = Signer::address_of(admin);
        assert!(exists<GuardianSet>(addr), E_NO_GUARDIAN_SET);
        let old = borrow_global_mut<GuardianSet>(addr);

        // Verify Governance Update.
        let update = parse(admin, payload);

        verify(&update, old);

        let GuardianUpdate {
            guardian_module,
            action,
            new_index,  
            guardians,
        } = update;

        old.index = old.index + 1;
        old.guardians = guardians;
    }

    public fun parse(admin: &signer, bytes: vector<u8>): GuardianUpdate {
        //let guardians = Table::new<u8, Guardian>();
        let guardians = Vector::empty<Guardian>();
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
            Vector::push_back(&mut guardians, Guardian {key:key});
            //Table::add(&mut guardians, guardian_len-1, Guardian { key:key});
            guardian_len = guardian_len - 1;
        };

        assert!(Vector::length(&mut bytes) == 0, E_REMAINING_BYTES);

        GuardianUpdate {
            guardian_module:    guardian_module,
            action:             action,
            new_index:          new_index,
            guardians:          guardians,
        }
    }
    
    public fun verify(update: &GuardianUpdate, previous: &GuardianSet){
        let (guardian_module, action) = (update.guardian_module, update.action);
        assert!(Vector::length(&guardian_module) == 32, 0);
        assert!(action == 0x02, 0); 
        assert!(update.new_index > previous.index, 0);
    }
}