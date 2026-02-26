;; Interface for handling VAAs from governance contract
;; We may not need to use this trait for abstraction or proxying,
;; but it's still useful to define the governance interface
(define-trait governance-trait
  (
    ;; Handle governance action #1: ContractUpgrade
    (contract-upgrade ((buff 8192)) (response (buff 32) uint))
    ;; Handle governance action #2: GuardianSetUpgrade
    (guardian-set-upgrade ((buff 8192) (list 30 (buff 64))) (response {
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
      result: {
        guardians-eth-addresses: (list 30 (buff 64)),
        guardians-public-keys: (list 30 (buff 64))
      }
    } uint))
    ;; Handle governance action #3: SetMessageFee
    (set-message-fee ((buff 8192)) (response uint uint))
    ;; Handle governance action #4: TransferFees
    (transfer-fees ((buff 8192)) (response {
      recipient: principal,
      amount: uint
    } uint))
  )
)
