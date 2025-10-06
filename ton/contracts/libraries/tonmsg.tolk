const ANSWER_BIT = 0x80000000;

@inline
fun sendChange<TPayload>(
    to: address, 
    opcode: uint32, 
    payload: TPayload,
) {
    val answerOpcode: uint32 = opcode | ANSWER_BIT;
    val reply = createMessage({
        bounce: false,
        value: 0,
        dest: to,
        body: (answerOpcode, payload)
    });
    reply.send(SEND_MODE_CARRY_ALL_REMAINING_MESSAGE_VALUE);
}

@inline
fun emit<TPayload>(payload: TPayload) {
    val extOutMsg = createExternalLogMessage({
        dest: createAddressNone(),
        body: payload
    });
    extOutMsg.send(SEND_MODE_REGULAR);
}

fun call<TPayload>(to: address, amount: coins, mode: uint8, payload: TPayload) {
    val msg = createMessage({
        bounce: true,
        value: amount,
        dest: to,
        body: payload
    });
    msg.send(mode);
}