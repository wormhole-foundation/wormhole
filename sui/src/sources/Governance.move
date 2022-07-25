

module Wormhole::Governance {
    use Wormhole::Deserialize;
    use Wormhole::VAA::{Self};//, get_payload};
    use std::vector;
    use sui::id;
    use sui::tx_context::{Self, TxContext};
    use sui::id::{ID, VersionedID};
    use sui::transfer;
    use sui::vec_map::{Self, VecMap};
    //use sui::option;
   
    const E_WRONG_GUARDIAN_LEN: u64 = 0x0;
    const E_REMAINING_BYTES: u64    = 0x1;

    struct GuardianUpdate has key{
        id:                 VersionedID,
        guardian_module:    vector<u8>,
        action:             u8,
        new_index:          u64,
        guardians:          VecMap<u8, Guardian>,
    }

    struct Guardian has key, store{
        id:        VersionedID,
        key:       vector<u8>,
    }

    struct GuardianSet has key {
        id:        VersionedID,
        index:     u64,
        guardians: VecMap<u8, Guardian>,
    }

    public fun init_guardian_set(ctx: &mut TxContext){
        let x = vec_map::empty();
        transfer::freeze_object(GuardianSet {
             id:        tx_context::new_id(ctx),
             index:     0,
             guardians: x,
        });
    }

    // Creates a new guardian set object with the given index. Takes an old guardian set as input.
    public fun update_guardian_set(vaa: vector<u8>, old: &GuardianSet, ctx: &mut TxContext) {
        // Verify VAA.
        let vaa = VAA::parse(vaa, ctx);

        //TODO: verify vaaa
        //VAA::verify(&vaa, &old);
        let payload = VAA::destroy(vaa);

        // Verify Governance Update.
        let update = parse(payload, ctx);
        verify(&update, old);

        let  GuardianUpdate {
            id,
            guardian_module,
            action,
            new_index,  
            guardians,
        } = update;
        id::delete(id);

        // New GuardianSet is an object output of new_guardian validation. Future messages
        // can re-use the object as a read-only input.
        transfer::freeze_object(GuardianSet {
            id:        tx_context::new_id(ctx),
            index:     old.index + 1,
            guardians: guardians,
        });
    }

    public fun parse(bytes: vector<u8>, ctx: &mut TxContext): GuardianUpdate {
        let guardians = vec_map::empty();
        let (guardian_module, bytes) = Deserialize::deserialize_vector(bytes, 32);
        //TODO: missing chain?
        let (action, bytes) = Deserialize::deserialize_u8(bytes);
        let (new_index, bytes) = Deserialize::deserialize_u64(bytes);
        let (guardian_len, bytes) = Deserialize::deserialize_u8(bytes);

        assert!((guardian_len as u64) < 19, E_WRONG_GUARDIAN_LEN);

        while ({
            spec {
                invariant guardian_len >= 0;
                invariant guardian_len < 19;
            };
            guardian_len > 0
        }) { 
            let (key, bytes) = Deserialize::deserialize_vector(bytes, 20);
            vec_map::insert(&mut guardians, guardian_len-1, Guardian { key:key, id:tx_context::new_id(ctx)});
            guardian_len = guardian_len - 1;
        };

        assert!((vector::length(&mut bytes) as u64) == (0 as u64), E_REMAINING_BYTES);

        GuardianUpdate {
            id:                 tx_context::new_id(ctx),
            guardian_module:    guardian_module,
            action:             action,
            new_index:          new_index,
            guardians:          guardians,
        }
    }
    
    public fun verify(update: &GuardianUpdate, previous: &GuardianSet) {
        let (guardian_module, action) = (update.guardian_module, update.action);
        assert!(vector::length(&guardian_module) == 32, 0);
        assert!(action == 0x02, 0);
        assert!(update.new_index > previous.index, 0);
    }
}
