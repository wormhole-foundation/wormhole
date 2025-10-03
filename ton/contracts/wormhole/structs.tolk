import "../libraries/errors";

const VERSION = 1;

struct GuardianSet {
    keys: map<uint8, uint256>
    count: uint8
    expirationTime: uint32
}

struct Signature {
    signature: bytes65 // 512 bits
    guardianIndex: uint8
}

struct VM {
    version: uint8
    timestamp: uint32
    nonce: uint32
    emitterChainId: uint16
    emitterAddress: uint256 // bytes32
    sequence: uint64
    consistencyLevel: uint8
    guardianSetIndex: uint32
    signatures: map<uint8, Signature>
    signaturesCount: uint8
    payload: cell
}

fun VM.parse(c: cell): (VM, uint256) {
    var cs = c.beginParse();
    val version = cs.loadUint(8);
    val guardianSetIndex = cs.loadUint(32);
    val signaturesCount = cs.loadUint(8);
    val signatures: map<uint8, Signature> = createMapFromLowLevelDict(cs.loadDict());
    val hash = cs.hash();
    val vm = VM{
        version: version,
        guardianSetIndex: guardianSetIndex,
        signatures: signatures,
        signaturesCount: signaturesCount,
        timestamp: cs.loadUint(32),
        nonce: cs.loadUint(32),
        emitterChainId: cs.loadUint(16),
        emitterAddress: cs.loadUint(256),
        sequence: cs.loadUint(64),
        consistencyLevel: cs.loadUint(8),
        payload: cs.loadRef(),
    };
    cs.assertEnd();
    return (vm, hash);
}

fun VM.hash(self): uint256 {
    return 0;
}

fun VM.validate(self) {
    assert(self.version == VERSION) throw ERROR_INVALID_VM_VERSION;
}

fun verifyVmSignatures(vmHash: uint256, guardianSet: GuardianSet, signatures: map<uint8, Signature>): bool {
    return true;
}