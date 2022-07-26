module Wormhole::Wormhole {
    use Wormhole::Governance;

    fun init(admin: &signer) {
        Governance::init_guardian_set(admin);
    }
}