;; Title: addr32
;; Version: final (CANNOT BE UPDATED)

;; This contract provides 32-byte addressing for the Stacks blockchain
;; A Stacks contract principal can be longer than 32 bytes, and some protocols can't handle that
;; We can generate a unique 32-byte address for any Stacks principal by hashing it
;; This allows us to use existing protocols unmodified

(define-constant ERR_INVALID_ADDRESS (err u901))

;; Registered principals
(define-map registry
  (buff 32)  ;; keccak256(principal)
  principal  ;; Stacks principal
)

;; @desc Get or register 32-byte address
(define-public (register (p principal))
  (if (is-standard p)
    ;; Address matches network, this is expected
    (inner-register p)
    ;; Address does not match network, need to support for unit tests
    (let ((addr32 (hash p)))
      (match (lookup addr32)
        val (ok {
          created: false,
          addr32: addr32
        })
        ERR_INVALID_ADDRESS))))

;; @desc Hash a Stacks principal to generate addr32
(define-read-only (hash (p principal))
  ;; `to-ascii?` cannot return errors for `principal` types
  (keccak256 (string-ascii-to-buff (unwrap-panic (to-ascii? p)))))

;; @desc Lookup Stacks principal for given addr32
(define-read-only (lookup (addr32 (buff 32)))
  (map-get? registry addr32))

;; @desc Lookup to see if Stacks principal is registered
(define-read-only (reverse-lookup (p principal))
  (let ((addr32 (hash p)))
    {
      registered: (is-some (map-get? registry addr32)),
      addr32: addr32
    }))

(define-private (string-ascii-to-buff (s (string-ascii 256)))
  (let ((cb (unwrap-panic (to-consensus-buff? s))))
    (unwrap-panic (slice? cb u5 (len cb)))))

;; @desc Bypass checks, used in unit tests
(define-private (inner-register (p principal))
  (let ((addr32 (hash p)))
    (ok {
      created: (map-insert registry addr32 p),
      addr32: addr32
    })))
