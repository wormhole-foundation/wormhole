;; title: wormhole-core-proxy-v2
;; version: v2
;; summary: Checks that `core` trait passed is valid and is active core contract
;; description:
;;
;; =============================
;; Proxy Summary
;; =============================
;;
;; We can't do a proper proxy in Clarity, as dynamic dispatch is limited in order to preserve decidability
;; The best we can do is have users of a third-party app query the state contract for the current active contract,
;; and pass it through this contract to validate it.
;;
;; This puts a burden on third-party developers to query the state contract and supply a value to users,
;; but allows them to avoid updating and re-deploying their contracts every time `wormhole-core` updates
;;
;; =============================
;; Proxy Dataflow
;; =============================
;;
;; -----------------------------
;; 1. Transaction Initiation
;; -----------------------------
;;   When a transaction is initiated, it must include the principal of the current active `wormhole-core` contract as an argument to the function call.
;;   This can be queried from the `wormhole-core-state` by the application and supplied to the user
;;
;; -----------------------------
;; 2. Third-party contract using Wormhole
;; -----------------------------
;;   The third-party contract must accept the trait as a function argument and pass it through to this contract (`wormhole-core-proxy-v2`)
;;   It is not necessary to check the trait argument here
;;
;; -----------------------------
;; 3. Trait Checker (THIS CONTRACT)
;; -----------------------------
;;   This will take the trait argument and check it against the currently active `wormhole-core` contract in `wormhole-core-state`.
;;   If it matches, it will call the the corresponding function in `wormhole-core`.
;;   If not, an error is returned
;;
;; -----------------------------
;; 4. Wormhole Core Contract
;; -----------------------------
;;   The last step is to call the currently active version of `wormhole-core` implementing `core-trait`

;;;; traits

(use-trait core-trait .wormhole-trait-core-v2.core-trait)

;;;; constants

;; No active wormhole-core contract found
(define-constant ERR_TRAIT_CHECK_NO_ACTIVE_CONTRACT (err u20001))
;; Trait does not match active contract
(define-constant ERR_TRAIT_CHECK_CONTRACT_MISMATCH (err u20002))

;;;; Public functions: Proxy for `wormhole-core`

(define-public (get-chain-id (core-contract <core-trait>))
  (begin
    (try! (check-active-wormhole-core-contract core-contract))
    (contract-call? core-contract get-chain-id)))

(define-public (get-message-fee (core-contract <core-trait>))
  (begin
    (try! (check-active-wormhole-core-contract core-contract))
    (contract-call? core-contract get-message-fee)))

(define-public (get-governance-contract (core-contract <core-trait>))
  (begin
    (try! (check-active-wormhole-core-contract core-contract))
    (contract-call? core-contract get-governance-contract)))

(define-public (parse-and-verify-vaa (core-contract <core-trait>) (vaa-bytes (buff 8192)))
  (begin
    (try! (check-active-wormhole-core-contract core-contract))
    (contract-call? core-contract parse-and-verify-vaa vaa-bytes)))

(define-public (post-message (core-contract <core-trait>) (payload (buff 8192)) (nonce uint) (consistency-level-opt (optional uint)))
  (begin
    (try! (check-active-wormhole-core-contract core-contract))
    (contract-call? core-contract post-message-via-proxy payload nonce consistency-level-opt contract-caller)))

(define-public (get-wormhole-address (core-contract <core-trait>) (p principal))
  (begin
    (try! (check-active-wormhole-core-contract core-contract))
    (contract-call? core-contract get-wormhole-address p)))

;;;; Read-only functions

(define-read-only (check-active-wormhole-core-contract (expected-core-contract <core-trait>))
  (let ((active-core-contract (unwrap! (contract-call? .wormhole-core-state get-active-wormhole-core-contract) ERR_TRAIT_CHECK_NO_ACTIVE_CONTRACT)))
    (asserts! (is-eq (contract-of expected-core-contract) active-core-contract) ERR_TRAIT_CHECK_CONTRACT_MISMATCH)
    (ok expected-core-contract)))