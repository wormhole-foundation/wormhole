module Wormhole::GuardianSet {
    use Sui::ID::VersionedID;
    use Sui::TxContext;
    use Sui::Transfer;
    use Wormhole::VAA;

    struct GuardianSet has key {
        id:        VersionedID,
        index:     u32,
        guardians: vector<vector<u8>>,
    }

    // Creates a new guardian set object with the given index. Takes an old guardian set as input.
    fun new_guardian_set(vaa: VAA::VAA, old: &GuardianSet, ctx: &mut TxContext) {
        use Wormhole::Governance::GuardianUpdate;

        // Verify VAA.
        let vaa = VAA::parse(vaa);
        VAA::verify(&vaa, &old);

        // Verify Governance Update.
        let update = GuardianUpdate::parse(vaa.payload);
        GuardianUpdate::verify(&update, &old.guardians););

        // New GuardianSet is an object output of new_guardian validation. Future messages
        // can re-use the object as a read-only input.
        Transfer::freeze_object(GuardianSet {
            id:        TxContext::new_id(ctx),
            index:     old.index + 1,
            guardians: update.guardians,
        });
    }
}
