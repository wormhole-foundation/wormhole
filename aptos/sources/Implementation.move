module Wormhole::Implementation{
    use 0x1::event::{Self};
    use 0x1::signer::{address_of};
    //use Wormhole::State::{WormholeMessage};
    use Wormhole::State::{nextSequence, setNextSequence, WormholeMessageHandle, publishMessage};
    
    
    
    // TODO: how to add fee? 
    //require(msg.value == messageFee(), "invalid fee");

    //sequence = useSequence(msg.sender);
    // emit log
    //emit LogMessagePublished(msg.sender, sequence, nonce, payload, consistencyLevel);

}



