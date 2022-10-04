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
/// The `aptos_framework::account::create_resource_account` function creates such an
/// account, and returns a newly created signer and a `SignerCapability`, which
/// is an affine resource (i.e. can be dropped but not copied). Holding a
/// `SignerCapability` (which can be stored) grants the ability to recover the
/// signer (which cannot be stored). Thus, the program will need to hold its own
/// `SignerCapability` in storage. It is crucial that this capability is kept
/// "securely", i.e. gated behind proper access control, which essentially means
/// in a struct with a private field. This capability is retrieved and stored in
/// the module's initializer.
///
/// So the strategy is as follows:
///
/// 1. An account calls `deploy_derived` with the bytecode and the address seed
/// The function will create the resource account and deploy the bytecode at the
/// resource account's address.  It will then temporarily lock up the
/// `SignerCapability` in `DeployingSignerCapability` together with the deployer
/// account's address.
///
/// Then there are two options:
/// 2.a. The module has an `init_module` entry point. This is a special function
/// that gets called by the runtime immediately after the module is deployed.
/// The only argument passed to this function is the module account's signer, in
/// this case the resource account itself. The resource account can call
/// `claim_signer_capability` and retrieve the signer capability to store it in
/// storage for later. This destroys the `DeployingSignerCapability`.
///
/// 2.b  The module has a custom initializer function. This might be necessary
/// if the initializer needs additional arguments, which is not supported by
/// `init_module`. This initializer will have to be called in a separate
/// transaction (since after deploying a module, it cannot be called in the same
/// transaction). The initializer may call `claim_signer_capability` which
/// destroys the `DeployingSignerCability` and extracts the `SignerCapability`
/// from it. Note that this can _only_ be called by the deployer account.
///
/// The `claim_signer_capability` function checks that it's called _either_ by
/// the resource account itself or the deployer of the resource account.
///
/// 3. After the `SignerCapability` is extracted, the program can now recover
/// the signer from it and store the capability in its own storage in a secure
/// resource type.
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

    public entry fun deploy_derived(
        deployer: &signer,
        metadata_serialized: vector<u8>,
        code: vector<vector<u8>>,
        seed: vector<u8>
    ) acquires DeployingSignerCapability {
        let deployer_address = signer::address_of(deployer);
        let resource = account::create_resource_address(&deployer_address, seed);
        let resource_signer: signer;
        if (exists<DeployingSignerCapability>(resource)) {
            // if the deploying signer capability already exists, it means that
            // the resource account hasn't claimed it. This code path allows the
            // deployer to upgrade the resource account's contract, but only
            // before the resource account is initialised.
            // You might think that this is a very niche use-case, but this
            // happened when trying to deploy wormhole to aptos testnet, as the
            // bytecode we published had been compiled with an older version of
            // the stdlib, and had native dependency issues. These are checked
            // lazily (i.e. at runtime, and not deployment time), which meant
            // that the contract was effectively broken, i.e. unable to
            // initialise itself, and therefore unable to upgrade.
            let deploying_cap = borrow_global<DeployingSignerCapability>(resource);
            resource_signer = account::create_signer_with_capability(&deploying_cap.signer_cap);
        } else {
            // if it doesn't exist, it means that either
            // a) the account hasn't been created yet at all
            // b) the account has already claimed the signer capability
            //
            // in the case of a), we just create it. In case of b), the account
            // creation will fail, since the resource account already exist,
            // effectively providing replay protection.
            let signer_cap: account::SignerCapability;
            (resource_signer, signer_cap) = account::create_resource_account(deployer, seed);
            move_to(&resource_signer, DeployingSignerCapability { signer_cap, deployer: deployer_address });
        };
        code::publish_package_txn(&resource_signer, metadata_serialized, code);
    }

    public fun claim_signer_capability(
        caller: &signer,
        resource: address
    ): account::SignerCapability acquires DeployingSignerCapability {
        assert!(exists<DeployingSignerCapability>(resource), E_NO_DEPLOYING_SIGNER_CAPABILITY);
        let DeployingSignerCapability { signer_cap, deployer } = move_from(resource);
        let caller_addr = signer::address_of(caller);
        assert!(
            caller_addr == deployer || caller_addr == resource,
            E_INVALID_DEPLOYER
        );
        signer_cap
    }
}
