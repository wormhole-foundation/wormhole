(use-trait ft-trait .trait-sip-010.sip-010-trait)
(define-trait flash-loan-user-trait
  (
    (execute (<ft-trait> uint (optional (buff 16))) (response bool uint))
  )
)