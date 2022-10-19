module wormhole::keccak256 {
    use sui::ecdsa;

    spec module {
        pragma verify=false;
    }

    public fun keccak256(bytes: vector<u8>): vector<u8> {
        ecdsa::keccak256(&bytes)
    }

    spec keccak256 {
        pragma opaque;
    }

}
