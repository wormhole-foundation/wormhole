module wormhole::keccak256 {
    use aptos_framework::aptos_hash;

    spec module {
        pragma verify=false;
    }

    public fun keccak256(bytes: vector<u8>): vector<u8> {
        aptos_hash::keccak256(bytes)
    }

    spec keccak256 {
        pragma opaque;
    }

}
