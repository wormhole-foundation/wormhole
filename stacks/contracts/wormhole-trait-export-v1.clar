;; This is for transferring the state of `wormhole-core` during updates
;; This does not cover `wormhole-core-state`, which cannot be updated
(define-trait export-trait
  (
    ;; Returns current state of `wormhole-core` and deactivates contract
    ;; Fails if caller is not successor contract
    (export-state () (response {
      active-guardian-set-id: (optional uint),
      previous-guardian-set: (optional {
        set-id: uint,
        expires-at: uint
      }),
      message-fee: uint,
    } uint))
  )
)
