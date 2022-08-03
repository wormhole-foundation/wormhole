module Wormhole::Implementation{
    

    public fun publishMessage(
        sender: &signer,
        nonce: u64, //should be u32
        payload: vector<u8>,
        consistencyLevel: u8, 
    ){

    }
        // TODO: how to add fee? 
        //require(msg.value == messageFee(), "invalid fee");

        //sequence = useSequence(msg.sender);
        // emit log
        //emit LogMessagePublished(msg.sender, sequence, nonce, payload, consistencyLevel);

}



