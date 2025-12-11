(define-constant err-invalid-length-version (err u1000))
(define-constant err-invalid-length-chain-length (err u1001))
(define-constant err-invalid-length-burn-spent (err u1002))
(define-constant err-invalid-length-consensus-hash (err u1003))
(define-constant err-invalid-length-parent-block-id (err u1004))
(define-constant err-invalid-length-tx-merkle-root (err u1005))
(define-constant err-invalid-length-state-index-root (err u1006))
(define-constant err-invalid-length-timestamp (err u1007))
(define-constant err-invalid-length-miner-signature (err u1008))
(define-constant err-invalid-length-signer-bitvec (err u1009))
(define-constant err-invalid-length-block-hash (err u1010))

(define-read-only (valid-signer-bitvec (bitvec (buff 506)))
	(let ((byte-length (buff-to-uint-be (unwrap-panic (as-max-len? (unwrap! (slice? bitvec u2 u6) false) u4)))))
		(is-eq (len bitvec) (+ byte-length u6))
	)
)

(define-read-only (block-header-hash
	(version (buff 1))
	(chain-length (buff 8))
	(burn-spent (buff 8))
	(consensus-hash (buff 20))
	(parent-block-id (buff 32))
	(tx-merkle-root (buff 32))
	(state-index-root (buff 32))
	(timestamp (buff 8))
	(miner-signature (buff 65))
	(signer-bitvec (buff 506))
	)
	(begin
		(asserts! (is-eq (len version) u1) err-invalid-length-version)
		(asserts! (is-eq (len chain-length) u8) err-invalid-length-chain-length)
		(asserts! (is-eq (len burn-spent) u8) err-invalid-length-burn-spent)
		(asserts! (is-eq (len consensus-hash) u20) err-invalid-length-consensus-hash)
		(asserts! (is-eq (len parent-block-id) u32) err-invalid-length-parent-block-id)
		(asserts! (is-eq (len tx-merkle-root) u32) err-invalid-length-tx-merkle-root)
		(asserts! (is-eq (len state-index-root) u32) err-invalid-length-state-index-root)
		(asserts! (is-eq (len timestamp) u8) err-invalid-length-timestamp)
		(asserts! (is-eq (len miner-signature) u65) err-invalid-length-miner-signature)
		(asserts! (valid-signer-bitvec signer-bitvec) err-invalid-length-signer-bitvec)
		(ok
			(sha512/256
				(concat version
				(concat chain-length
				(concat burn-spent
				(concat consensus-hash
				(concat parent-block-id
				(concat tx-merkle-root
				(concat state-index-root
				(concat timestamp
				(concat miner-signature signer-bitvec)))))))))
			)
		)
	)
)

(define-read-only (block-id-header-hash (block-hash (buff 32)) (consensus-hash (buff 20)))
	(begin
		(asserts! (is-eq (len block-hash) u32) err-invalid-length-block-hash)
		(asserts! (is-eq (len consensus-hash) u20) err-invalid-length-consensus-hash)
		(ok (sha512/256 (concat block-hash consensus-hash)))
	)
)

(define-read-only (string-ascii-to-buffer (str (string-ascii 16000)))
	(unwrap-panic (slice? (unwrap-panic (to-consensus-buff? str)) u5 (+ (len str) u5)))
)
