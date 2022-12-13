/// This module implements upgradeability for the wormhole contract.
///
/// Contract upgrades are authorised by governance, which means that performing
/// an upgrade requires a governance VAA signed by a supermajority of the
/// wormhole guardians.
///
/// Upgrades are performed in a commit-reveal scheme, where submitting the VAA
/// authorises a particular contract hash. Then in a subsequent transaction, the
/// bytecode is uploaded, and if the hash of the bytecode matches the committed
/// hash, then the upgrade proceeds.
///
/// This two-phase process has the advantage that even if the bytecode can't be
/// upgraded to for whatever reason, the governance VAA won't be possible to
/// replay in the future, since the commit transaction replay protects it.
///
/// Additionally, there is an optional migration step that may include one-off
/// logic to be executed after the upgrade. This has to be done in a separate
/// transaction, because the transaction that uploads bytecode cannot execute
/// it.
module wormhole::contract_upgrade {
    use std::vector;
    use aptos_framework::code;
    use wormhole::deserialize;
    use wormhole::cursor;
    use wormhole::vaa;
    use wormhole::state;
    use wormhole::keccak256::keccak256;

    /// "Core" (left padded)
    const CORE: vector<u8> = x"00000000000000000000000000000000000000000000000000000000436f7265";

    const E_UPGRADE_UNAUTHORIZED: u64 = 0;
    const E_UNEXPECTED_HASH: u64 = 1;
    const E_INVALID_MODULE: u64 = 2;
    const E_INVALID_ACTION: u64 = 3;
    const E_INVALID_TARGET: u64 = 4;
    const E_NOT_MIGRATING: u64 = 5;

    /// The `UpgradeAuthorized` type in the global storage represents the fact
    /// there is an ongoing approved upgrade.
    /// When the upgrade is finalised in `upgrade`, this object is deleted.
    struct UpgradeAuthorized has key {
        hash: vector<u8>
    }

    struct Hash has drop {
        hash: vector<u8>
    }

    public fun get_hash(hash: &Hash): vector<u8> {
        hash.hash
    }

    fun parse_payload(payload: vector<u8>): Hash {
        let cur = cursor::init(payload);
        let target_module = deserialize::deserialize_vector(&mut cur, 32);

        assert!(target_module == CORE, E_INVALID_MODULE);

        let action = deserialize::deserialize_u8(&mut cur);
        assert!(action == 0x01, E_INVALID_ACTION);

        let chain = deserialize::deserialize_u16(&mut cur);
        assert!(chain == state::get_chain_id(), E_INVALID_TARGET);

        let hash = deserialize::deserialize_vector(&mut cur, 32);

        cursor::destroy_empty(cur);

        Hash { hash }
    }

// -----------------------------------------------------------------------------
// Commit

    public fun submit_vaa(
        vaa: vector<u8>
    ): Hash acquires UpgradeAuthorized {
        let vaa = vaa::parse_and_verify(vaa);
        vaa::assert_governance(&vaa);
        vaa::replay_protect(&vaa);

        let hash = parse_payload(vaa::destroy(vaa));
        authorize_upgrade(&hash);
        hash
    }

    public entry fun submit_vaa_entry(vaa: vector<u8>) acquires UpgradeAuthorized {
        submit_vaa(vaa);
    }

    fun authorize_upgrade(hash: &Hash) acquires UpgradeAuthorized {
        let wormhole = state::wormhole_signer();
        if (exists<UpgradeAuthorized>(@wormhole)) {
            // TODO(csongor): here we're dropping the upgrade hash, in case an
            // upgrade fails for some reason. Should we emit a log or something?
            let UpgradeAuthorized { hash: _ } = move_from<UpgradeAuthorized>(@wormhole);
        };
        move_to(&wormhole, UpgradeAuthorized { hash: hash.hash });
    }

    #[test_only]
    public fun authorized_hash(): vector<u8> acquires UpgradeAuthorized {
        let u = borrow_global<UpgradeAuthorized>(@wormhole);
        u.hash
    }

// -----------------------------------------------------------------------------
// Reveal

    public entry fun upgrade(
        metadata_serialized: vector<u8>,
        code: vector<vector<u8>>
    ) acquires UpgradeAuthorized {
        assert!(exists<UpgradeAuthorized>(@wormhole), E_UPGRADE_UNAUTHORIZED);
        let UpgradeAuthorized { hash } = move_from<UpgradeAuthorized>(@wormhole);

        // we compute the hash of hashes of the metadata and the bytecodes.
        // the aptos framework appears to perform no validation of the metadata,
        // so we check it here too.
        let c = copy code;
        vector::reverse(&mut c);
        let a = keccak256(metadata_serialized);
        while (!vector::is_empty(&c)) vector::append(&mut a, keccak256(vector::pop_back(&mut c)));
        assert!(keccak256(a) == hash, E_UNEXPECTED_HASH);

        let wormhole = state::wormhole_signer();
        code::publish_package_txn(&wormhole, metadata_serialized, code);

        // allow migration to be run.
        if (!exists<Migrating>(@wormhole)) {
            move_to(&wormhole, Migrating {});
        }
    }

// -----------------------------------------------------------------------------
// Migration

    struct Migrating has key {}

    public fun is_migrating(): bool {
        exists<Migrating>(@wormhole)
    }

    public entry fun migrate() acquires Migrating {
        assert!(exists<Migrating>(@wormhole), E_NOT_MIGRATING);
        let Migrating { } = move_from<Migrating>(@wormhole);

        // NOTE: put any one-off migration logic here.
        // Most upgrades likely won't need to do anything, in which case the
        // rest of this function's body may be empty.
        // Make sure to delete it after the migration has gone through
        // successfully.
        // WARNING: the migration does *not* proceed atomically with the
        // upgrade (as they are done in separate transactions).
        // If the nature of your migration absolutely requires the migration to
        // happen before certain other functionality is available, then guard
        // that functionality with `assert!(!is_migrating())` (from above).
    }
}

#[test_only]
module wormhole::contract_upgrade_test {
    use wormhole::contract_upgrade;
    use wormhole::wormhole;

    const UPGRADE_VAA: vector<u8> = x"010000000001000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a30100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000020b10360000000000000000000000000000000000000000000000000000000000436f7265010016d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

    fun setup() {
        let aptos_framework = std::account::create_account_for_test(@aptos_framework);
        std::timestamp::set_time_has_started_for_testing(&aptos_framework);
        wormhole::init_test(
            22,
            1,
            x"0000000000000000000000000000000000000000000000000000000000000004",
            x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe",
            0
        );
    }

    #[test]
    public fun test_contract_upgrade_authorize() {
        setup();

        contract_upgrade::submit_vaa(UPGRADE_VAA);
        let expected_hash = x"d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

        assert!(contract_upgrade::authorized_hash() == expected_hash, 0);
    }

    #[test]
    #[expected_failure(abort_code = 0x6407, location = 0x1::table)]
    public fun test_contract_upgrade_double() {
        setup();

        // make sure we can't replay a VAA
        contract_upgrade::submit_vaa(UPGRADE_VAA);
        contract_upgrade::submit_vaa(UPGRADE_VAA);
    }

    #[test]
    #[expected_failure(abort_code = 4, location = contract_upgrade)]
    public fun test_contract_upgrade_wrong_chain() {
        setup();

        let eth_upgrade = x"01000000000100d46215cd004a6a9d50114d31efdcba3e769dc559a7550c5e90618cacc5808d1e52d982d68e98369946fcfa46d47ade3ad88d9e4c2634a2d1a564b7aecb33e0d7000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000000c88e420000000000000000000000000000000000000000000000000000000000436f72650100029fc26e02f9bc648d48b7076571f1790049b2049d0101d4c52419c9ab8134ecb6";
        contract_upgrade::submit_vaa(eth_upgrade);
    }

}
