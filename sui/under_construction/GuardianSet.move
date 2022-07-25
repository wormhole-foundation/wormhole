module Wormhole::GuardianSet {
    use sui::id::VersionedID;
    use sui::tx_context::{Self, TxContext};
    use sui::transfer;
    use Wormhole::VAA::{Self, get_payload};
    use Wormhole::Governance::{parse, verify};

    struct Guardian has key, store {
        id:        VersionedID,
        key:       vector<u8>,
    }

    struct GuardianSet has key {
        id:        VersionedID,
        index:     u64,
        guardians: vector<Guardian>,
    }

    // Creates a new guardian set object with the given index. Takes an old guardian set as input.
    public fun new_guardian_set(vaa: vector<u8>, old: &GuardianSet, ctx: &mut TxContext) {
        // Verify VAA.
        let vaa = VAA::parse(vaa, ctx);

        //TODO: verify vaaa
        //VAA::verify(&vaa, &old);

        // Verify Governance Update.
        let update = parse(get_payload(&mut vaa), ctx);
        verify(&update, old.guardians);

        // New GuardianSet is an object output of new_guardian validation. Future messages
        // can re-use the object as a read-only input.
        transfer::freeze_object(GuardianSet {
            id:        TxContext::new_id(ctx),
            index:     old.index + 1,
            guardians: update.guardians,
        });
    }
}
