
// the struct uses a 32-bit opcode prefix for message identification
struct (0x1CE51423) PublishMessage {
    queryId: uint64
    nonce: uint32
    consistencyLevel: uint8
    payload: cell
    tail: cell
}

struct (0xABFC7DB9) ParseAndVerifyVM {
    queryId: uint64
    encodedVM: cell
}
