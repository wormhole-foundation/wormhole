;; Trait which can be implemented by any cross-chain protocol (Wormhole, Axelar, etc.) to send messages
;; This trait is necessary in order for NTT manager to be protocol-agnostic
(define-trait send-trait
  (
    ;; Send message to network
    ;; The params to this function are a combination of Axelar's `call-contract` and Wormhole's `post-message`
    ;; Additional params can be supported if needed on other networks by encoding them in the byte buffer
    (protocol-agnostic-send (
		    (buff 64000)                  ;; Payload (all protocols)
        (optional (string-ascii 20))  ;; Destination chain (Axelar)
        (optional (string-ascii 128)) ;; Destination contract address (Axelar)
		    (optional uint)               ;; Nonce (Wormhole)
		    (optional uint))              ;; Consistency level (Wormhole)
	  (response uint uint))
  )
)