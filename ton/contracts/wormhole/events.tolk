
const EVENT_PUBLISH_MESSAGE = stringCrc32("event::wormhole::publish_message");

struct (0x50ACEA3E) PublishMessageEvent {
    sender: address
    sequence: uint64
    nonce: uint32
    payload: cell
    consistencyLevel: uint8
}