
const EVENT_PUBLISH_MESSAGE = stringCrc32("event::message_published");

struct (0xA237A664) MessagePublishedEvent {
    sender: address
    sequence: uint64
    nonce: uint32
    payload: cell
    consistencyLevel: uint8
}