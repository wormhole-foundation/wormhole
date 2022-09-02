module wormhole::contract_upgrade {
    use std::aptos_hash;
    use std::vector;
    use aptos_framework::code;
    use wormhole::deserialize;
    use wormhole::cursor;
    use wormhole::vaa;
    use wormhole::state;

    const E_UPGRADE_UNAUTHORIZED: u64 = 0;
    const E_UNEXPECTED_HASH: u64 = 1;
    const E_INVALID_MODULE: u64 = 2;
    const E_INVALID_ACTION: u64 = 3;
    const E_INVALID_TARGET: u64 = 4;

    // TODO(csongor): document how this works
    struct UpgradeAuthorized has key {
        hash: vector<u8>
    }

    struct Hash {
        hash: vector<u8>
    }

    // TODO(csongor): maybe a parse and verify...?
    fun parse_payload(payload: vector<u8>): Hash {
        let cur = cursor::init(payload);
        let target_module = deserialize::deserialize_vector(&mut cur, 32);

        // TODO(csongor): refactor this (like deserialize_module_magic or something)
        let expected_module = x"00000000000000000000000000000000000000000000000000000000436f7265"; // Core
        assert!(target_module == expected_module, E_INVALID_MODULE);

        let action = deserialize::deserialize_u8(&mut cur);
        assert!(action == 0x01, E_INVALID_ACTION);

        let chain = deserialize::deserialize_u16(&mut cur);
        assert!(chain == state::get_chain_id(), E_INVALID_TARGET);

        let hash = deserialize::deserialize_vector(&mut cur, 32);

        cursor::destroy_empty(cur);

        Hash { hash }
    }

    public entry fun submit_vaa(
        vaa: vector<u8>
    ) acquires UpgradeAuthorized {
        let vaa = vaa::parse_and_verify(vaa);
        vaa::assert_governance(&vaa);
        vaa::replay_protect(&vaa);

        authorize_upgrade(parse_payload(vaa::destroy(vaa)));
    }

    fun authorize_upgrade(hash: Hash) acquires UpgradeAuthorized {
        let Hash { hash } = hash;
        let wormhole = state::wormhole_signer();
        if (exists<UpgradeAuthorized>(@wormhole)) {
            // TODO(csongor): here we're dropping the upgrade hash, in case an
            // upgrade fails for some reason. Should we emit a log or something?
            let UpgradeAuthorized { hash: _ } = move_from<UpgradeAuthorized>(@wormhole);
        };
        move_to(&wormhole, UpgradeAuthorized { hash });
    }

    #[test_only]
    public fun authorized_hash(): vector<u8> acquires UpgradeAuthorized {
        let u = borrow_global<UpgradeAuthorized>(@wormhole);
        u.hash
    }

    public entry fun upgrade(
        metadata_serialized: vector<u8>,
        code: vector<vector<u8>>
    ) acquires UpgradeAuthorized {
        assert!(exists<UpgradeAuthorized>(@wormhole), E_UPGRADE_UNAUTHORIZED);
        let UpgradeAuthorized { hash } = move_from<UpgradeAuthorized>(@wormhole);

        let c = copy code;
        vector::reverse(&mut c);
        let a = vector::empty<u8>();
        while (!vector::is_empty(&c)) vector::append(&mut a, vector::pop_back(&mut c));
        assert!(aptos_hash::keccak256(a) == hash, E_UNEXPECTED_HASH);

        let wormhole = state::wormhole_signer();
        code::publish_package_txn(&wormhole, metadata_serialized, code);
    }
}

#[test_only]
module wormhole::contract_upgrade_test {
    use wormhole::contract_upgrade;
    use wormhole::wormhole;

    const UPGRADE_VAA: vector<u8> = x"010000000001000da16466429ee8ffb09b90ca90db8326d20cfeeae0542da9dcaaad641a5aca2d6c1fe33a5970ca84fd0ff5e6d29ef9e40404eb1a8892b509f085fc725b9e23a30100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000020b10360000000000000000000000000000000000000000000000000000000000436f7265010016d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

    fun setup(aptos_framework: signer, user: &signer) {
        std::account::create_account_for_test(@aptos_framework);
        std::timestamp::set_time_has_started_for_testing(&aptos_framework);
        wormhole::init_test(
            user,
            22,
            1,
            x"0000000000000000000000000000000000000000000000000000000000000004",
            x"beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
        );
    }

    #[test(aptos_framework = @aptos_framework, user=@0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b)]
    public fun test_contract_upgrade_authorize(aptos_framework: signer, user: &signer) {
        setup(aptos_framework, user);

        contract_upgrade::submit_vaa(UPGRADE_VAA);
        let expected_hash = x"d8f30e4a345ea0fa5df11daac4e1866ee368d253209cf9eda012d915a2db09e6";

        assert!(contract_upgrade::authorized_hash() == expected_hash, 0);
    }

    #[test(aptos_framework = @aptos_framework, user=@0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b)]
    #[expected_failure(abort_code = 0x6407)]
    public fun test_contract_upgrade_double(aptos_framework: signer, user: &signer) {
        setup(aptos_framework, user);

        // make sure we can't replay a VAA
        contract_upgrade::submit_vaa(UPGRADE_VAA);
        contract_upgrade::submit_vaa(UPGRADE_VAA);
    }

    #[test(aptos_framework = @aptos_framework, user=@0x277fa055b6a73c42c0662d5236c65c864ccbf2d4abd21f174a30c8b786eab84b)]
    #[expected_failure(abort_code = 4)]
    public fun test_contract_upgrade_wrong_chain(aptos_framework: signer, user: &signer) {
        setup(aptos_framework, user);

        let eth_upgrade = x"01000000000100d46215cd004a6a9d50114d31efdcba3e769dc559a7550c5e90618cacc5808d1e52d982d68e98369946fcfa46d47ade3ad88d9e4c2634a2d1a564b7aecb33e0d7000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000000c88e420000000000000000000000000000000000000000000000000000000000436f72650100029fc26e02f9bc648d48b7076571f1790049b2049d0101d4c52419c9ab8134ecb6";
        contract_upgrade::submit_vaa(eth_upgrade);
    }

}
