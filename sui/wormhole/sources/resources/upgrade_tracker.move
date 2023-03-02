module wormhole::upgrade_tracker {
    use sui::table::{Self, Table};
    use sui::tx_context::{TxContext};

    const E_CURRENT_IMPLEMENTATION_REQUIRED: u64 = 0;

    struct UpgradeTracker has store {
        global_version: u64,
        gated_versions: Table<vector<u8>, u64>
    }

    public fun new(version: u64, ctx: &mut TxContext): UpgradeTracker {
        UpgradeTracker {
            global_version: version,
            gated_versions: table::new(ctx)
        }
    }

    public fun current(self: &UpgradeTracker): u64 {
        self.global_version
    }

    public fun add(self: &mut UpgradeTracker, method_label: vector<u8>) {
        table::add(&mut self.gated_versions, method_label, self.global_version)
    }

    /// This method will abort if the version for a particular method is not
    /// up-to-date with the implementation's version that this value is checked
    /// against.
    ///
    /// For example, if the version for a method `foobar` is known to be `1` and
    /// the current implementation version is `2`, this method will succeed
    /// because the implementation version is at least `1` in order for `foobar`
    /// to work. So if someone were to use a dependency which is implementation
    /// version `1`, `foobar` will also work because that implementation's
    /// version is at least `1` as well.
    ///
    /// But if `require_current_version` were invoked for `foobar` when the
    /// package's implementation version is now `2`, then implementation version
    /// `1` will fail because `1` is less than `2`, which is the known version
    /// for `foobar`.
    public fun assert_current(
        self: &UpgradeTracker,
        method_label: vector<u8>,
        impl_version: u64
    ) {
        assert!(
            impl_version >= current_for(self, method_label),
            E_CURRENT_IMPLEMENTATION_REQUIRED
        );
    }

    /// At `commit_upgrade`, use this method to update the tracker's knowledge
    /// of the current implementation version, which is obtained from the
    /// `UpgradeCap` using `package::version`.
    public fun update_global(self: &mut UpgradeTracker, new_version: u64) {
        self.global_version = new_version;
    }

    /// Once the global version is updated via `commit_upgrade`, if there is a
    /// particular method that has a breaking change, use this method to uptick
    /// that method to the current version so `assert_current` will abort when
    /// the implementation is a stale version.
    public fun require_current_version(
        self: &mut UpgradeTracker,
        method_label: vector<u8>
    ) {
        let gated = table::borrow_mut(&mut self.gated_versions, method_label);
        *gated = self.global_version;
    }

    fun current_for(self: &UpgradeTracker, method_label: vector<u8>): u64 {
        *table::borrow(&self.gated_versions, method_label)
    }
}

#[test_only]
module wormhole::upgrade_tracker_test {
    // TODO
}
