module wormhole::keccak256 {
    use sui::crypto::Self;

    spec module {
        pragma verify=false;
    }

    public fun keccak256(bytes: vector<u8>): vector<u8> {
        crypto::keccak256(bytes)
    }

    spec keccak256 {
        pragma opaque;
    }

}