export const TON_CHAIN_ID = 62;

export const Opcodes = {
    OP_PUBLISH_MESSAGE: 0x1ce51423,
    OP_PARSE_AND_VERIFY_VM: 0x051679d3,
    OP_SEND_COMMENT: 0x222a627e,
    OP_RELAY_COMMENT: 0x327587b5,
    ANSWER_BIT: 0x80000000,
};

export const Events = {
    EVENT_MESSAGE_PUBLISHED: 0xa237a664,
    EVENT_VAA_VALIDATED_BY_CORE: 0x00000001,
};

export const toAnswer = (opcode: number) => (opcode | Opcodes.ANSWER_BIT) >>> 0;