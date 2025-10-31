;; These are the functions required by third-party contracts using the Wormhold protocol
;; If we implement a proxy contract in front of wormhole-core (to allow for easy updates),
;; we will need to use this trait to proxy these functions
(define-trait core-trait
  (
    ;; Getters
    ;; Ideally these would not return `result` types, but currently trait-defined functions must do this 
    (get-chain-id () (response (buff 2) uint))
    (get-message-fee () (response uint uint))
    (get-governance-contract () (response {
      chain-id: uint,
      address: (buff 32),
    } uint))
    ;; Parse and Verify cryptographic validity of a VAA
    (parse-and-verify-vaa ((buff 8192)) (response {
      vaa: {
        version: uint,
        guardian-set-id: uint,
        signatures-len: uint ,
        signatures: (list 30 { guardian-id: uint, signature: (buff 65) }),
        timestamp: uint,
        nonce: uint,
        emitter-chain: uint,
        emitter-address: (buff 32),
        sequence: uint,
        consistency-level: uint,
        payload: (buff 8192),
      },
      vaa-body-hash: (buff 32)
    } uint))
    ;; Emit message for Guardians to observe
    ;; Provided in this trait in case user wants to implement their own proxy logic
    ;; NOTE: May charge sender a fee to send message
    (post-message ((buff 8192) uint (optional uint)) (response {
      emitter-principal: principal,
      emitter: (buff 32),
      nonce: uint,                ;; Must fit into `u32`
      sequence: uint,             ;; Must fit into `u64`
      consistency-level: uint,    ;; Must fit into `u8`
      payload: (buff 8192)
    } uint))
    ;; Emit message for Guardians to observe
    ;; Can only called by *OUR* proxy contract!
    ;; NOTE: May charge sender a fee to send message
    (post-message-via-proxy ((buff 8192) uint (optional uint) principal) (response {
      emitter-principal: principal,
      emitter: (buff 32),
      nonce: uint,                ;; Must fit into `u32`
      sequence: uint,             ;; Must fit into `u64`
      consistency-level: uint,    ;; Must fit into `u8`
      payload: (buff 8192)
    } uint))
    ;; Get or generate new "Wormhole address" for a Stacks `prrincipal` that can be used in Wormhole messages
    ;; Addresses in the Wormhole protocol are limited to 32 bytes, but a Stacks `principal` can be longer than this
    ;; NOTE: May charge sender a fee to generate new address
    (get-wormhole-address (principal) (response {
      created: bool,
      wormhole-address: (buff 32)
    } uint))
  )
)
