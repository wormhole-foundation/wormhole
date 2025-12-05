;; clarity-stacks
;; Check if a Stacks transaction has been mined.
;; Only works for Nakamoto blocks.

(define-constant err-invalid-length-txid (err u2000))
(define-constant err-proof-too-short (err u2001))
(define-constant err-block-header-too-short (err u2005))
(define-constant err-invalid-block-height (err u2002))
(define-constant err-block-height-header-mismatch (err u2003))
(define-constant err-merkle-proof-invalid (err u2004))

(define-constant merkle-path-leaf-tag 0x00)
(define-constant merkle-path-node-tag 0x01)

(define-read-only (tagged-hash (tag (buff 1)) (data (buff 64)))
	(sha512/256 (concat tag data))
)

(define-read-only (is-bit-set (val uint) (bit uint))
	(> (bit-and val (bit-shift-left u1 bit)) u0)
)

(define-read-only (merkle-leaf-hash (data (buff 32)))
	(tagged-hash merkle-path-leaf-tag data)
)

(define-private (inner-merkle-proof-verify (ctr uint) (state { path: uint, root-hash: (buff 32), proof-hashes: (list 14 (buff 32)), tree-depth: uint, cur-hash: (buff 32), verified: bool}))
  (let ((path (get path state))
        (is-left (is-bit-set path ctr))
        (proof-hashes (get proof-hashes state))
        (cur-hash (get cur-hash state))
        (root-hash (get root-hash state))
        (h1 (if is-left (unwrap-panic (element-at proof-hashes ctr)) cur-hash))
        (h2 (if is-left cur-hash (unwrap-panic (element-at proof-hashes ctr))))
        (next-hash (tagged-hash merkle-path-node-tag (concat h1 h2)))
        (is-verified (and (is-eq (+ u1 ctr) (len proof-hashes)) (is-eq next-hash root-hash)))
		)
    	(merge state { cur-hash: next-hash, verified: is-verified})
	)
)

;; Note that the hashes in the proof must be tagged hashes.
;; Do not put TXIDs in the proof directly, they must first be
;; hashed with (merkle-leaf-hash).
;; Returns (ok true) if the proof is valid, or an error if not.
(define-read-only (verify-merkle-proof (txid (buff 32)) (merkle-root (buff 32)) (proof { tx-index: uint, hashes: (list 14 (buff 32)), tree-depth: uint}))
	(if (> (get tree-depth proof) (len (get hashes proof)))
		err-proof-too-short
		(ok (asserts! (get verified
			(fold inner-merkle-proof-verify
				(unwrap-panic (slice? (list u0 u1 u2 u3 u4 u5 u6 u7 u8 u9 u10 u11 u12 u13) u0 (get tree-depth proof)))
				{
					path: (+ (pow u2 (get tree-depth proof)) (get tx-index proof)),
					root-hash: merkle-root, proof-hashes: (get hashes proof),
					cur-hash: (tagged-hash merkle-path-leaf-tag txid),
					tree-depth: (get tree-depth proof),
					verified: false
				}
			)) err-merkle-proof-invalid)
		)
	)
)

(define-read-only (get-block-info-header-hash? (stx-height uint))
	(get-stacks-block-info? header-hash stx-height)
)

;; Returns (ok true) if the transaction was mined.
(define-read-only (was-tx-mined-compact (txid (buff 32)) (proof { tx-index: uint, hashes: (list 14 (buff 32)), tree-depth: uint}) (tx-block-height uint) (block-header-without-signer-signatures (buff 712)))
	(let (
		(target-header-hash (unwrap! (get-block-info-header-hash? tx-block-height) err-invalid-block-height))
		(tx-merkle-root (unwrap-panic (as-max-len? (unwrap! (slice? block-header-without-signer-signatures u69 u101) err-block-header-too-short) u32)))
		(header-hash (sha512/256 block-header-without-signer-signatures))
		)
		(asserts! (is-eq (len txid) u32) err-invalid-length-txid)
		;; It is fine to compare header hash because the consensus hash is part
		;; of the header in Nakamoto.
		(asserts! (is-eq header-hash target-header-hash) err-block-height-header-mismatch)
		(verify-merkle-proof txid tx-merkle-root proof)
	)
)
