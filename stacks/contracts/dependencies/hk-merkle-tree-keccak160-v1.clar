;; Title: hiro-merkle-tree-keccak160
;; Version: v1

(define-read-only (keccak160 (bytes (buff 1024)))
    (unwrap-panic (as-max-len? (unwrap-panic (slice? (keccak256 bytes) u0 u20)) u20)))

(define-read-only (buff-20-to-uint (bytes (buff 20)))
    (buff-to-uint-be (unwrap-panic (as-max-len? (unwrap-panic (slice? bytes u0 u15)) u16))))

(define-read-only (hash-leaf (bytes (buff 255)))
    (keccak160 (concat 0x00 bytes)))

(define-read-only (hash-nodes (node-1 (buff 20)) (node-2 (buff 20)))
    (let ((uint-1 (buff-20-to-uint node-1))
          (uint-2 (buff-20-to-uint node-2))
          (sequence (if (< uint-2 uint-1) 
            (concat (concat 0x01 node-2) node-1)
            (concat (concat 0x01 node-1) node-2))))
    (keccak160 sequence)))

(define-read-only (check-proof (root-hash (buff 20)) (leaf (buff 255)) (path (list 255 (buff 20))))
    (let ((hashed-leaf (hash-leaf leaf))
          (computed-root-hash (fold hash-path path hashed-leaf)))
        (is-eq root-hash computed-root-hash)))

(define-private (hash-path (entry (buff 20)) (acc (buff 20)))
    (hash-nodes entry acc))
