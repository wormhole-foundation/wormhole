/// This is a deployer module that enables deploying a package to an autonomous
/// account, or a "resource account".
///
/// By default, when an account publishes a package, the modules get deployed at
/// the deployer account's address, and retains authority over the package.
/// For applications that want to guard their upgradeability by some other
/// mechanism (such as decentralized governance), this setup is inadequate.
///
/// The solution is to generate an autonomous account whose signer is controlled
/// by the runtime, as opposed to a private key (think program-derived addresses
/// in Solana). The package is then deployed at this address, guaranteeing that
/// effectively the program can upgrade itself in whatever way it wishes, and no
/// one else can.
///
/// The `aptos_framework::account` module provides a way to generate such
/// resource accounts, where the account's pubkey is derived from the
/// transaction signer's account hashed together with some seed. In pseudocode:
///
///     resource_address = sha3_256(tx_sender + seed)
///
/// The `aptos_framework::create_resource_account` function creates such an
/// account, and returns a newly created signer and a `SignerCapability`, which
/// is an affine resource (i.e. can be dropped but not copied). Holding a
/// `SignerCapability` (which can be stored) grants the ability to recover the
/// signer (which cannot be stored). Thus, the program will need to hold its own
/// `SignerCapability` in storage. It is crucial that this capability is kept
/// "securely", i.e. gated behind proper access control, which essentially means
/// in a struct with a private field. There's a chicken-and-egg problem here,
/// because the new program needs to be able to create and store that struct,
/// but it just got deployed with the newly created signer, and it's not
/// possible to execute bytecode in the same transaction that deployed the
/// bytecode.
///
/// So the strategy is as follows:
///
/// 1. An account calls `deploy_derived` with the bytecode and the address seed
/// (TODO(csongor): add the seed as an argument). The function will create the
/// resource account and deploy the bytecode at the resource account's address.
/// It will then temporarily lock up the `SignerCapability` in
/// `DeployingSignerCapability` together with the deployer account's address.
///
/// 2. In a separate transaction, the deployer account calls the initialization
/// method of the newly deployed program. In the initialization method, the
/// program may call `claim_signer_capability` which destroys the
/// `DeployingSignerCability` and extracts the `SignerCapability` from it.
/// Note that this can _only_ be called by the deployer account.
///
/// 3. After the `SignerCapability` is extracted, the program can now recover
/// the signer from it and store the capability in its own storage in a secure
/// resource type.
///
///
/// Note that the fact that `SignerCapability` has no copy ability means that
/// it's guaranteed to be globally unique for a given resource account (since
/// the function that creates it can only be called once as it implements replay
/// protection). Thanks to this, as long as the deployed program is successfully
/// initialized and stores its signer capability, we can be sure that only the
/// program can authorize its own upgrades.
///
module deployer::deployer {
    use aptos_framework::account;
    use aptos_framework::code;
    use aptos_framework::signer;

    const E_NO_DEPLOYING_SIGNER_CAPABILITY: u64 = 0;
    const E_INVALID_DEPLOYER: u64 = 1;

    /// Resource for temporarily holding on to the signer capability of a newly
    /// deployed program before the program claims it.
    struct DeployingSignerCapability has key {
        signer_cap: account::SignerCapability,
        deployer: address,
    }

    public entry fun deploy_derived(deployer: &signer, metadata_serialized: vector<u8>, code: vector<vector<u8>>, seed: vector<u8>) {
        let (wormhole, signer_cap) = account::create_resource_account(deployer, seed);
        let deployer = signer::address_of(deployer);
        move_to(&wormhole, DeployingSignerCapability { signer_cap, deployer });
        code::publish_package_txn(&wormhole, metadata_serialized, code);
    }

    public fun claim_signer_capability(
        user: &signer,
        resource: address
    ): account::SignerCapability acquires DeployingSignerCapability {
        assert!(exists<DeployingSignerCapability>(resource), E_NO_DEPLOYING_SIGNER_CAPABILITY);
        let DeployingSignerCapability { signer_cap, deployer } = move_from(resource);
        assert!(signer::address_of(user) == deployer, E_INVALID_DEPLOYER);
        signer_cap
    }
}
