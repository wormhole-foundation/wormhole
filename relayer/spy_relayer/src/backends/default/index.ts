import { Backend, Relayer, Listener } from "../definitions"
import { TokenBridgeListener } from "./listener"
import { TokenBridgeRelayer } from "./relayer"

/** Payload version 1 token bridge listener and relayer backened */
const backend: Backend = {
    relayer:  new TokenBridgeRelayer(),
    listener: new TokenBridgeListener(),

}

export default backend