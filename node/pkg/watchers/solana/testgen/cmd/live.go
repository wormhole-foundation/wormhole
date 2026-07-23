package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/solana/testgen"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Mainnet Wormhole program ids (see pkg/watchers/solana/client_test.go, shim_test.go).
var (
	coreProgram = solana.MustPublicKeyFromBase58("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth")
	shimProgram = solana.MustPublicKeyFromBase58("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX")
	altProgram  = solana.MustPublicKeyFromBase58("AddressLookupTab1e1111111111111111111111111")
)

// Instruction discriminators the watcher keys off of (client.go, close_event.go, shim.go).
const (
	postMessageID           = 0x01
	postMessageUnreliableID = 0x08
	closePostedMessageID    = 0x09
	postMessageMinAccounts  = 8

	defaultWormholescanBase = "https://api.wormholescan.io/api/v1"

	// close_posted_message transactions are rare and clustered in time (all issued around
	// slot ~425,726,140 / 2026-06-11). Paging back from the tip is slow, so the close search
	// is anchored just before this known cluster transaction (which carries 22 close
	// instructions) and seeds it directly.
	defaultCloseAnchor = "3FNV3mYTfQxAP8tk1mPh2hfa7qWoJ85MfmQTiFtrS7TiJAWQqPvxdAupDHWiJKQ4efboznzCtQxAGDJEuGYrVcaM"

	rpcTimeout = 60 * time.Second
	rpcRetries = 5
)

// shimPostMessageDiscriminator is the 8-byte prefix of a shim post_message instruction.
var shimPostMessageDiscriminator = mustHex("d63264d12622074c")

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// errShortfall signals that collection did not reach its per-category targets.
var errShortfall = errors.New("shortfall")

type liveConfig struct {
	rpc                string
	postMessage        int
	shim               int
	close              int
	pageSize           int
	maxPages           int
	sleep              time.Duration
	wormholescanBase   string
	verifyWormholescan bool
	keepMissingAccount bool
	closeAnchor        string
}

// detection is one recognized Wormhole message inside a transaction.
type detection struct {
	category       string // post_message | shim | close
	location       string // outer | inner
	kind           string // postmessage | postmessageunreliable | ""
	messageAccount *solana.PublicKey
}

type collector struct {
	ctx    context.Context
	client *rpc.Client
	http   *http.Client
	cfg    liveConfig

	records []*testgen.Bundle
	seen    map[solana.Signature]bool
	counts  map[string]int
	stats   map[string]int
}

// ------------------------------------------------------------------------- RPC

// withRetry runs fn, retrying with linear back-off on any error (covers 429/server-busy).
func (c *collector) withRetry(name string, fn func() error) error {
	var last error
	for attempt := 0; attempt < rpcRetries; attempt++ {
		if err := fn(); err != nil {
			last = err
			backoff := time.Duration(2*(attempt+1)) * time.Second
			if backoff > 8*time.Second {
				backoff = 8 * time.Second
			}
			time.Sleep(backoff) //nolint:forbidigo // dev-only tool, needs RPC back-off
			continue
		}
		return nil
	}
	return fmt.Errorf("%s failed after %d attempts: %w", name, rpcRetries, last)
}

func (c *collector) getSignatures(program solana.PublicKey, before solana.Signature) ([]*rpc.TransactionSignature, error) {
	limit := c.cfg.pageSize
	opts := &rpc.GetSignaturesForAddressOpts{Limit: &limit, Commitment: rpc.CommitmentFinalized}
	if !before.IsZero() {
		opts.Before = before
	}
	var out []*rpc.TransactionSignature
	err := c.withRetry("getSignaturesForAddress", func() error {
		rctx, cancel := context.WithTimeout(c.ctx, rpcTimeout)
		defer cancel()
		var e error
		out, e = c.client.GetSignaturesForAddressWithOpts(rctx, program, opts)
		return e
	})
	return out, err
}

func (c *collector) getTransaction(sig solana.Signature) (*rpc.GetTransactionResult, error) {
	maxVer := uint64(0)
	opts := &rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		Commitment:                     rpc.CommitmentFinalized,
		MaxSupportedTransactionVersion: &maxVer,
	}
	var out *rpc.GetTransactionResult
	err := c.withRetry("getTransaction", func() error {
		rctx, cancel := context.WithTimeout(c.ctx, rpcTimeout)
		defer cancel()
		var e error
		out, e = c.client.GetTransaction(rctx, sig, opts)
		return e
	})
	return out, err
}

func (c *collector) getAccountInfo(key solana.PublicKey) (*rpc.Account, error) {
	opts := &rpc.GetAccountInfoOpts{Encoding: solana.EncodingBase64, Commitment: rpc.CommitmentFinalized}
	var out *rpc.GetAccountInfoResult
	err := c.withRetry("getAccountInfo", func() error {
		rctx, cancel := context.WithTimeout(c.ctx, rpcTimeout)
		defer cancel()
		var e error
		out, e = c.client.GetAccountInfoWithOpts(rctx, key, opts)
		return e
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	return out.Value, nil
}

// wormholescanHasTx is a best-effort WormholeScan confirmation for a Solana signature.
func (c *collector) wormholescanHasTx(signature string) bool {
	u := fmt.Sprintf("%s/operations?txHash=%s", c.cfg.wormholescanBase, url.QueryEscape(signature))
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	req.Header.Set("accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false
	}
	var asObj map[string]json.RawMessage
	if json.Unmarshal(body, &asObj) == nil {
		for _, key := range []string{"data", "operations", "items", "results"} {
			var arr []json.RawMessage
			if v, ok := asObj[key]; ok && json.Unmarshal(v, &arr) == nil && len(arr) > 0 {
				return true
			}
		}
		return false
	}
	var asArr []json.RawMessage
	return json.Unmarshal(body, &asArr) == nil && len(asArr) > 0
}

// ---------------------------------------------------------------- classification

// accountKeys returns the static account keys plus any loaded (ALT) addresses, in index order.
func accountKeys(tx *solana.Transaction, meta *rpc.TransactionMeta) []solana.PublicKey {
	keys := append([]solana.PublicKey{}, tx.Message.AccountKeys...)
	keys = append(keys, meta.LoadedAddresses.Writable...)
	keys = append(keys, meta.LoadedAddresses.ReadOnly...)
	return keys
}

func classifyInstruction(program solana.PublicKey, data []byte, accounts []uint16, keys []solana.PublicKey, location string) *detection {
	if program.Equals(coreProgram) && len(data) > 0 {
		ident := data[0]
		if (ident == postMessageID || ident == postMessageUnreliableID) && len(accounts) >= postMessageMinAccounts {
			kind := "postmessage"
			if ident == postMessageUnreliableID {
				kind = "postmessageunreliable"
			}
			// The second account in a well-formed post_message is the message account.
			var msgAcct *solana.PublicKey
			if idx := int(accounts[1]); idx < len(keys) {
				k := keys[idx]
				msgAcct = &k
			}
			return &detection{category: "post_message", location: location, kind: kind, messageAccount: msgAcct}
		}
		if ident == closePostedMessageID {
			return &detection{category: "close", location: location}
		}
	}
	if program.Equals(shimProgram) && bytes.HasPrefix(data, shimPostMessageDiscriminator) {
		return &detection{category: "shim", location: location}
	}
	return nil
}

// classify walks top-level and inner instructions, returning every recognized message.
func classify(tx *solana.Transaction, meta *rpc.TransactionMeta) []detection {
	keys := accountKeys(tx, meta)
	programOf := func(inst solana.CompiledInstruction) solana.PublicKey {
		if idx := int(inst.ProgramIDIndex); idx < len(keys) {
			return keys[idx]
		}
		return solana.PublicKey{}
	}

	var detections []detection
	for _, inst := range tx.Message.Instructions {
		if det := classifyInstruction(programOf(inst), inst.Data, inst.Accounts, keys, "outer"); det != nil {
			detections = append(detections, *det)
		}
	}
	for _, group := range meta.InnerInstructions {
		for _, inst := range group.Instructions {
			if det := classifyInstruction(programOf(inst), inst.Data, inst.Accounts, keys, "inner"); det != nil {
				detections = append(detections, *det)
			}
		}
	}

	// A shim publication IS a core post_message_unreliable: the tx calls the shim contract,
	// which then CPIs the core bridge. So a tx containing a shim instruction is a shim tx --
	// its core post_message is the shim's own CPI and must not be double-counted as a
	// standalone "post_message". Keep shim (and close); drop the post_message detections.
	hasShim := false
	for _, d := range detections {
		if d.category == "shim" {
			hasShim = true
			break
		}
	}
	if hasShim {
		kept := detections[:0]
		for _, d := range detections {
			if d.category != "post_message" {
				kept = append(kept, d)
			}
		}
		detections = kept
	}
	return detections
}

// ---------------------------------------------------------------- record build

func accountResponse(pubkey solana.PublicKey, acc *rpc.Account) testgen.AccountInfoResponse {
	return testgen.AccountInfoResponse{
		Pubkey:     pubkey,
		Owner:      acc.Owner,
		Lamports:   acc.Lamports,
		Data:       acc.Data.GetBinary(),
		Executable: acc.Executable,
		RentEpoch:  acc.RentEpoch,
	}
}

// fetchPostMessageAccount best-effort fetches the post_message account so the bundle can replay.
func (c *collector) fetchPostMessageAccount(det detection) []testgen.AccountInfoResponse {
	if det.messageAccount == nil {
		return nil
	}
	acc, err := c.getAccountInfo(*det.messageAccount)
	time.Sleep(c.cfg.sleep) //nolint:forbidigo // RPC rate limiting
	if err != nil {
		fmt.Fprintf(os.Stderr, "  message account %s fetch failed: %v\n", det.messageAccount, err)
		return nil
	}
	if acc == nil {
		return nil
	}
	return []testgen.AccountInfoResponse{accountResponse(*det.messageAccount, acc)}
}

// fetchLookupTableAccounts fetches the ALT accounts a versioned transaction references. Returns
// (accounts, true) when all are present and ALT-owned (empty for a legacy tx), or (nil, false)
// if any is cleaned up / wrong-owner (the caller then skips the transaction).
func (c *collector) fetchLookupTableAccounts(tx *solana.Transaction) ([]testgen.AccountInfoResponse, bool) {
	var out []testgen.AccountInfoResponse
	for _, lut := range tx.Message.GetAddressTableLookups() {
		acc, err := c.getAccountInfo(lut.AccountKey)
		time.Sleep(c.cfg.sleep) //nolint:forbidigo // RPC rate limiting
		if err != nil {
			fmt.Fprintf(os.Stderr, "  lookup table %s fetch failed: %v\n", lut.AccountKey, err)
			return nil, false
		}
		if acc == nil || !acc.Owner.Equals(altProgram) {
			return nil, false
		}
		out = append(out, accountResponse(lut.AccountKey, acc))
	}
	return out, true
}

// buildBundle emits a testgen Bundle with its source signature. The category/kind/location are
// folded into Name; the harness fills `expected` on first run.
func buildBundle(det detection, res *rpc.GetTransactionResult, tx *solana.Transaction, meta *rpc.TransactionMeta, signature solana.Signature, accounts []testgen.AccountInfoResponse) *testgen.Bundle {
	sig := signature.String()
	nameKind := ""
	if det.kind != "" {
		nameKind = "_" + det.kind
	}
	return &testgen.Bundle{
		Name:         fmt.Sprintf("live_%s%s_%s_%s", det.category, nameKind, det.location, sig[:8]),
		Slot:         res.Slot,
		Contract:     coreProgram,
		ShimContract: shimProgram,
		Transaction:  tx,
		Meta:         meta,
		Accounts:     accounts, // nil marshals to null, matching static bundles
		Signature:    sig,
	}
}

// normalizeMeta drops the large arrays the harness does not need (pre/postBalances, logMessages).
func normalizeMeta(meta *rpc.TransactionMeta) {
	meta.PreBalances = nil
	meta.PostBalances = nil
	meta.LogMessages = nil
}

// ------------------------------------------------------------------ collection

// tryCollect fetches, classifies, and (if it matches the wanted category with quota left)
// records one transaction. Returns true iff a bundle was appended.
func (c *collector) tryCollect(signature solana.Signature, entryErr interface{}, category string, target int, ctxLabel string) bool {
	if c.seen[signature] {
		return false
	}
	if entryErr != nil {
		c.stats["failed_tx_skipped"]++
		return false
	}

	res, err := c.getTransaction(signature)
	time.Sleep(c.cfg.sleep) //nolint:forbidigo // RPC rate limiting
	if err != nil {
		fmt.Fprintf(os.Stderr, "  getTransaction(%s) failed: %v\n", signature.String()[:8], err)
		return false
	}
	if res == nil || res.Transaction == nil {
		return false
	}
	tx, err := res.Transaction.GetTransaction()
	if err != nil || tx == nil {
		fmt.Fprintf(os.Stderr, "  decode(%s) failed: %v\n", signature.String()[:8], err)
		return false
	}
	meta := res.Meta
	if meta == nil {
		return false
	}
	if meta.Err != nil {
		c.stats["failed_tx_skipped"]++
		return false
	}
	if tx.Message.IsVersioned() {
		c.stats["versioned"]++
	}

	chosen := (*detection)(nil)
	for _, d := range classify(tx, meta) {
		if d.category == category && c.counts[category] < target {
			dd := d
			chosen = &dd
			break
		}
	}
	if chosen == nil {
		return false
	}

	if c.cfg.verifyWormholescan {
		if !c.wormholescanHasTx(signature.String()) {
			return false
		}
		time.Sleep(c.cfg.sleep) //nolint:forbidigo // RPC rate limiting
	}

	// A versioned tx only replays if its ALT accounts are still on-chain (the watcher
	// re-fetches them). Fetch them; skip the tx if any is cleaned up.
	altAccounts, ok := c.fetchLookupTableAccounts(tx)
	if !ok {
		c.stats["alt_missing_skipped"]++
		return false
	}

	var accounts []testgen.AccountInfoResponse
	if chosen.category == "post_message" {
		accounts = c.fetchPostMessageAccount(*chosen)
		if len(accounts) == 0 && !c.cfg.keepMissingAccount {
			c.stats["post_message_account_missing"]++
			return false
		}
	}
	accounts = append(accounts, altAccounts...)

	normalizeMeta(meta)
	c.records = append(c.records, buildBundle(*chosen, res, tx, meta, signature, accounts))
	c.seen[signature] = true

	kindPart := ""
	if chosen.kind != "" {
		kindPart = "/" + chosen.kind
	}
	fmt.Fprintf(os.Stderr, "collected %s %s (%s%s): %s\n", chosen.category, ctxLabel, chosen.location, kindPart, signature)
	return true
}

// collectBulk walks signatures for program and harvests up to target of one category. Paging
// starts newest-first, or from startBefore (exclusive) when set. Any seed signatures are
// processed first so a known-good tx is captured even though `before` is exclusive of it.
func (c *collector) collectBulk(program solana.PublicKey, category string, target int, startBefore solana.Signature, seed []solana.Signature) {
	for _, s := range seed {
		if c.counts[category] >= target {
			return
		}
		if c.tryCollect(s, nil, category, target, fmt.Sprintf("%d/%d", c.counts[category]+1, target)) {
			c.counts[category]++
		}
	}

	before := startBefore
	for page := 0; page < c.cfg.maxPages; page++ {
		if c.counts[category] >= target {
			return
		}
		sigs, err := c.getSignatures(program, before)
		if err != nil {
			fmt.Fprintf(os.Stderr, "getSignaturesForAddress(%s) page %d failed: %v\n", program.String()[:8], page+1, err)
			return
		}
		if len(sigs) == 0 {
			return
		}
		before = sigs[len(sigs)-1].Signature
		for _, entry := range sigs {
			if c.counts[category] >= target {
				return
			}
			if c.tryCollect(entry.Signature, entry.Err, category, target, fmt.Sprintf("%d/%d", c.counts[category]+1, target)) {
				c.counts[category]++
			}
		}
	}
}

func validateBundle(b *testgen.Bundle) error {
	if b.Name == "" || b.Signature == "" || b.Transaction == nil || b.Meta == nil {
		return errors.New("bundle missing required fields")
	}
	if !b.Contract.Equals(coreProgram) {
		return errors.New("contract is not the core program")
	}
	if len(b.Accounts) > 0 && !b.Accounts[0].Owner.Equals(coreProgram) {
		// A well-formed post_message account is owned by the core program; warn only.
		fmt.Fprintf(os.Stderr, "WARNING: %s message account owner %s != core\n", b.Name, b.Accounts[0].Owner)
	}
	return nil
}

// collectLiveBundles scans on-chain for the three message flows and returns the collected
// bundles, sorted. It returns errShortfall (with the bundles it did gather) if any per-category
// target was not met.
func collectLiveBundles(ctx context.Context, cfg liveConfig) ([]*testgen.Bundle, error) {
	c := &collector{
		ctx:    ctx,
		client: rpc.New(cfg.rpc),
		http:   &http.Client{Timeout: 20 * time.Second},
		cfg:    cfg,
		seen:   map[solana.Signature]bool{},
		counts: map[string]int{"post_message": 0, "shim": 0, "close": 0},
		stats:  map[string]int{"versioned": 0, "alt_missing_skipped": 0, "failed_tx_skipped": 0, "post_message_account_missing": 0},
	}

	fmt.Fprintf(os.Stderr, "Collecting live Solana Wormhole transactions: post_message=%d shim=%d close=%d\n", cfg.postMessage, cfg.shim, cfg.close)

	// Scan one category at a time rather than bucketing every type from a shared signature
	// stream: the message types are not uniformly present on-chain, so scanning each
	// separately lets us choose where to look. post_message and shim are frequent, so we take
	// them from the tip; close events are rare and clustered, so that scan is anchored near
	// their historical cluster instead of paging back from the tip.
	if cfg.postMessage > 0 {
		fmt.Fprintf(os.Stderr, "Scanning core %s for post_message (from tip) ...\n", coreProgram)
		c.collectBulk(coreProgram, "post_message", cfg.postMessage, solana.Signature{}, nil)
	}
	if cfg.shim > 0 {
		fmt.Fprintf(os.Stderr, "Scanning shim %s for shim (from tip) ...\n", shimProgram)
		c.collectBulk(shimProgram, "shim", cfg.shim, solana.Signature{}, nil)
	}
	if cfg.close > 0 {
		var anchor solana.Signature
		var seed []solana.Signature
		where := "from tip"
		if cfg.closeAnchor != "" {
			a, err := solana.SignatureFromBase58(cfg.closeAnchor)
			if err != nil {
				return nil, fmt.Errorf("invalid --close-anchor: %w", err)
			}
			anchor = a
			seed = []solana.Signature{a}
			where = fmt.Sprintf("anchored at %s...", cfg.closeAnchor[:8])
		}
		fmt.Fprintf(os.Stderr, "Scanning core %s for close (%s) ...\n", coreProgram, where)
		c.collectBulk(coreProgram, "close", cfg.close, anchor, seed)
	}

	for _, b := range c.records {
		if err := validateBundle(b); err != nil {
			return nil, fmt.Errorf("bundle %q: %w", b.Name, err)
		}
	}

	sort.Slice(c.records, func(i, j int) bool {
		if c.records[i].Name != c.records[j].Name {
			return c.records[i].Name < c.records[j].Name
		}
		return c.records[i].Slot < c.records[j].Slot
	})

	fmt.Fprintf(os.Stderr,
		"collected %d bundles (post_message=%d shim=%d close=%d); versioned=%d skipped: alt-missing=%d failed=%d post-message-account-missing=%d\n",
		len(c.records), c.counts["post_message"], c.counts["shim"], c.counts["close"],
		c.stats["versioned"], c.stats["alt_missing_skipped"], c.stats["failed_tx_skipped"], c.stats["post_message_account_missing"])

	var shortfall []string
	for _, cat := range []string{"post_message", "shim", "close"} {
		target := map[string]int{"post_message": cfg.postMessage, "shim": cfg.shim, "close": cfg.close}[cat]
		if c.counts[cat] < target {
			shortfall = append(shortfall, fmt.Sprintf("%s (%d/%d)", cat, c.counts[cat], target))
		}
	}
	if len(shortfall) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: did not reach target for: %v. Increase --max-pages or --page-size.\n", shortfall)
		return c.records, errShortfall
	}
	return c.records, nil
}
