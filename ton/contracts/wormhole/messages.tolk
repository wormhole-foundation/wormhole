
// the struct uses a 32-bit opcode prefix for message identification
const OP_PUBLISH_MESSAGE = stringCrc32("op::wormhole::publish_message");
const OP_PARSE_AND_VERIFY_VM = stringCrc32("op::wormhole::parse_and_verify_VM");

struct (0x1CE51423) PublishMessage {
    queryId: uint64
    nonce: uint32
    consistencyLevel: uint8
    payload: cell
    tail: cell
}

// opcode = OP_PUBLISH_MESSAGE | ANSWER_BIT
struct (0x9CE51423) PublishMessageAnswer {
    queryId: uint64
    tail: cell
}

struct (0x051679D3) ParseAndVerifyVM {
    queryId: uint64
    encodedVM: cell
    tail: cell
}

// opcode = OP_PARSE_AND_VERIFY_VM | ANSWER_BIT
struct (0x851679D3) ParseAndVerifyVMAnswer {
    queryId: uint64
    valid: bool
    encodedVaa: cell
    tail: cell
}
