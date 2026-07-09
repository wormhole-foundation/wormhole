#!/usr/bin/env python3
"""
Collect Ethereum Wormhole Core Bridge LogMessagePublished EVM logs.

Output schema matches the provided sample:
[
  {
    "comment": "...",
    "messageSent": true,
    "hash": "<Wormhole VAA body digest, no 0x>",
    "Sender": "0x...",
    "Sequence": 123,
    "Nonce": 0,
    "Payload": "<base64>",
    "ConsistencyLevel": 1,
    "Raw": { ... raw eth_getLogs/Etherscan-style log ... }
  }
]

Requirements:
  python -m pip install pycryptodome

Example:
  export ETH_RPC_URL="https://YOUR_ETHEREUM_RPC"
  python collect_wormhole_eth_logs.py --rpc "$ETH_RPC_URL" --count 200 --out real_out.json

By default the script checks WormholeScan's operations endpoint for each transaction hash.
Use --skip-wormholescan if WormholeScan is rate-limiting or unavailable and you want to
collect directly from Ethereum logs only.
"""
from __future__ import annotations

import argparse
import base64
import datetime as dt
import json
import random
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Dict, Iterable, List, Optional, Tuple

try:
    from Crypto.Hash import keccak
except Exception as exc:  # pragma: no cover
    raise SystemExit(
        "Missing dependency: pycryptodome. Install with: python -m pip install pycryptodome"
    ) from exc

ETH_WORMHOLE_CHAIN_ID = 2
CORE_BRIDGE = "0x98f3c9e6e3face36baad05fe09d375ef1464288b"
LOG_MESSAGE_PUBLISHED_TOPIC = "0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2"
DEFAULT_WORMHOLESCAN_BASE = "https://api.wormholescan.io/api/v1"


def _json_post(url: str, payload: Dict[str, Any], timeout: int = 30) -> Dict[str, Any]:
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"content-type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8"))


def rpc_call(rpc_url: str, method: str, params: List[Any], timeout: int = 30) -> Any:
    body = {"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
    out = _json_post(rpc_url, body, timeout=timeout)
    if "error" in out:
        raise RuntimeError(f"RPC error for {method}: {out['error']}")
    return out["result"]


def get_block_number(rpc_url: str) -> int:
    return int(rpc_call(rpc_url, "eth_blockNumber", []), 16)


def get_block(rpc_url: str, block_number: int) -> Dict[str, Any]:
    return rpc_call(rpc_url, "eth_getBlockByNumber", [hex(block_number), False])


def block_timestamp(rpc_url: str, block_number: int, cache: Dict[int, int]) -> int:
    if block_number not in cache:
        block = get_block(rpc_url, block_number)
        cache[block_number] = int(block["timestamp"], 16)
    return cache[block_number]


def find_first_block_at_or_after_timestamp(rpc_url: str, target_ts: int, latest_block: int) -> int:
    lo, hi = 0, latest_block
    cache: Dict[int, int] = {}
    while lo < hi:
        mid = (lo + hi) // 2
        ts = block_timestamp(rpc_url, mid, cache)
        if ts < target_ts:
            lo = mid + 1
        else:
            hi = mid
    return lo


def eth_get_logs(rpc_url: str, from_block: int, to_block: int) -> List[Dict[str, Any]]:
    filt = {
        "fromBlock": hex(from_block),
        "toBlock": hex(to_block),
        "address": CORE_BRIDGE,
        "topics": [LOG_MESSAGE_PUBLISHED_TOPIC],
    }
    return rpc_call(rpc_url, "eth_getLogs", [filt], timeout=60)


def http_json_get(url: str, timeout: int = 30) -> Any:
    req = urllib.request.Request(url, headers={"accept": "application/json"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8"))


def wormholescan_has_tx(base_url: str, tx_hash: str) -> bool:
    """Best-effort WormholeScan check. Treat a non-empty response as confirmation."""
    url = f"{base_url.rstrip('/')}/operations?txHash={urllib.parse.quote(tx_hash)}"
    try:
        data = http_json_get(url, timeout=20)
    except (urllib.error.URLError, urllib.error.HTTPError, TimeoutError, json.JSONDecodeError):
        return False

    def contains_tx(obj: Any) -> bool:
        if isinstance(obj, str):
            return obj.lower() == tx_hash.lower()
        if isinstance(obj, list):
            return any(contains_tx(x) for x in obj)
        if isinstance(obj, dict):
            return any(contains_tx(v) for v in obj.values())
        return False

    if contains_tx(data):
        return True
    if isinstance(data, list):
        return len(data) > 0
    if isinstance(data, dict):
        for key in ("data", "operations", "items", "results"):
            value = data.get(key)
            if isinstance(value, list) and len(value) > 0:
                return True
    return False


def keccak256(data: bytes) -> bytes:
    h = keccak.new(digest_bits=256)
    h.update(data)
    return h.digest()


def decode_log_message_published_data(data_hex: str) -> Tuple[int, int, int, bytes]:
    """Decode ABI data for LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel)."""
    h = data_hex[2:] if data_hex.startswith("0x") else data_hex
    if len(h) < 256:
        raise ValueError("log data too short for LogMessagePublished")

    sequence = int(h[0:64], 16)
    nonce = int(h[64:128], 16)
    payload_offset = int(h[128:192], 16)
    consistency_level = int(h[192:256], 16)

    len_start = payload_offset * 2
    if len(h) < len_start + 64:
        raise ValueError("log data missing dynamic bytes length")
    payload_len = int(h[len_start : len_start + 64], 16)
    payload_start = len_start + 64
    payload_hex = h[payload_start : payload_start + payload_len * 2]
    if len(payload_hex) != payload_len * 2:
        raise ValueError("log data has truncated payload")

    return sequence, nonce, consistency_level, bytes.fromhex(payload_hex)


def vaa_body_digest(
    timestamp: int,
    nonce: int,
    emitter_chain: int,
    emitter_address_topic: str,
    sequence: int,
    consistency_level: int,
    payload: bytes,
) -> str:
    topic = emitter_address_topic[2:] if emitter_address_topic.startswith("0x") else emitter_address_topic
    emitter_address = bytes.fromhex(topic)
    if len(emitter_address) != 32:
        raise ValueError("emitter address must be the 32-byte indexed sender topic")
    body = (
        int(timestamp).to_bytes(4, "big")
        + int(nonce).to_bytes(4, "big")
        + int(emitter_chain).to_bytes(2, "big")
        + emitter_address
        + int(sequence).to_bytes(8, "big")
        + int(consistency_level).to_bytes(1, "big")
        + payload
    )
    return keccak256(keccak256(body)).hex()


def raw_log_normalized(log: Dict[str, Any]) -> Dict[str, Any]:
    return {
        "address": log["address"].lower(),
        "topics": [t.lower() for t in log["topics"]],
        "data": log["data"].lower(),
        "blockNumber": log["blockNumber"].lower(),
        "transactionHash": log["transactionHash"].lower(),
        "transactionIndex": log["transactionIndex"].lower(),
        "blockHash": log["blockHash"].lower(),
        "logIndex": log["logIndex"].lower(),
        "removed": bool(log.get("removed", False)),
    }


def build_record(log: Dict[str, Any], timestamp: int) -> Dict[str, Any]:
    raw = raw_log_normalized(log)
    if raw["address"] != CORE_BRIDGE:
        raise ValueError("wrong contract address")
    if not raw["topics"] or raw["topics"][0] != LOG_MESSAGE_PUBLISHED_TOPIC:
        raise ValueError("wrong event topic")
    if len(raw["topics"]) < 2:
        raise ValueError("missing indexed sender topic")

    sequence, nonce, consistency_level, payload = decode_log_message_published_data(raw["data"])
    sender_topic = raw["topics"][1]
    sender = "0x" + sender_topic[-40:]
    digest = vaa_body_digest(
        timestamp,
        nonce,
        ETH_WORMHOLE_CHAIN_ID,
        sender_topic,
        sequence,
        consistency_level,
        payload,
    )
    return {
        "comment": (
            "real: Ethereum Wormhole Core LogMessagePublished; "
            f"tx {raw['transactionHash']} log {int(raw['logIndex'], 16)}"
        ),
        "messageSent": True,
        "hash": digest,
        "Sender": sender,
        "Sequence": sequence,
        "Nonce": nonce,
        "Payload": base64.b64encode(payload).decode("ascii"),
        "ConsistencyLevel": consistency_level,
        "Raw": raw,
    }


def validate_record(record: Dict[str, Any]) -> None:
    top = {"comment", "messageSent", "hash", "Sender", "Sequence", "Nonce", "Payload", "ConsistencyLevel", "Raw"}
    raw = {"address", "topics", "data", "blockNumber", "transactionHash", "transactionIndex", "blockHash", "logIndex", "removed"}
    missing_top = top - record.keys()
    if missing_top:
        raise ValueError(f"record missing fields: {sorted(missing_top)}")
    missing_raw = raw - record["Raw"].keys()
    if missing_raw:
        raise ValueError(f"record.Raw missing fields: {sorted(missing_raw)}")
    if record["Raw"]["address"].lower() != CORE_BRIDGE:
        raise ValueError("record.Raw.address is not Wormhole Core Bridge")
    if record["Raw"]["topics"][0].lower() != LOG_MESSAGE_PUBLISHED_TOPIC:
        raise ValueError("record.Raw.topics[0] is not LogMessagePublished")
    if record["Sender"].lower() != ("0x" + record["Raw"]["topics"][1][-40:].lower()):
        raise ValueError("Sender does not match indexed sender topic")
    seq, nonce, consistency, payload = decode_log_message_published_data(record["Raw"]["data"])
    if seq != record["Sequence"] or nonce != record["Nonce"] or consistency != record["ConsistencyLevel"]:
        raise ValueError("decoded values do not match record fields")
    if base64.b64decode(record["Payload"]) != payload:
        raise ValueError("Payload base64 does not match raw log payload bytes")


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Collect Ethereum Wormhole LogMessagePublished logs")
    p.add_argument("--rpc", required=True, help="Ethereum JSON-RPC URL")
    p.add_argument("--count", type=int, default=200, help="number of logs to collect")
    p.add_argument("--out", default="real_out.json", help="output JSON path")
    p.add_argument("--days", type=int, default=730, help="history window in days; default is roughly 2 years")
    p.add_argument("--window", type=int, default=5000, help="random eth_getLogs block window size")
    p.add_argument("--max-attempts", type=int, default=2000, help="max random windows to query")
    p.add_argument("--seed", type=int, default=None, help="random seed for reproducible sampling")
    p.add_argument("--start-block", type=int, default=None, help="override start block")
    p.add_argument("--end-block", type=int, default=None, help="override end block")
    p.add_argument("--wormholescan-base", default=DEFAULT_WORMHOLESCAN_BASE, help="WormholeScan API base URL")
    p.add_argument("--skip-wormholescan", action="store_true", help="do not verify tx hashes with WormholeScan")
    p.add_argument("--sleep", type=float, default=0.05, help="delay between accepted logs/API checks")
    return p.parse_args()


def main() -> int:
    args = parse_args()
    rng = random.Random(args.seed)

    latest = args.end_block if args.end_block is not None else get_block_number(args.rpc)
    latest_ts = int(get_block(args.rpc, latest)["timestamp"], 16)
    start_ts = latest_ts - args.days * 24 * 60 * 60
    start_block = args.start_block
    if start_block is None:
        start_block = find_first_block_at_or_after_timestamp(args.rpc, start_ts, latest)

    if start_block >= latest:
        raise SystemExit("start block is not before latest/end block")

    print(
        f"Sampling Wormhole logs from blocks {start_block}..{latest} "
        f"({args.days} days, target {args.count} records)",
        file=sys.stderr,
    )

    records: List[Dict[str, Any]] = []
    seen: set[Tuple[str, str]] = set()
    block_ts_cache: Dict[int, int] = {}
    attempts = 0
    window = max(1, args.window)

    while len(records) < args.count and attempts < args.max_attempts:
        attempts += 1
        lo = rng.randint(start_block, max(start_block, latest - window))
        hi = min(latest, lo + window)

        try:
            logs = eth_get_logs(args.rpc, lo, hi)
        except Exception as exc:
            print(f"attempt {attempts}: eth_getLogs {lo}-{hi} failed: {exc}", file=sys.stderr)
            if window > 100:
                window = max(100, window // 2)
                print(f"reducing --window to {window}", file=sys.stderr)
            time.sleep(0.5)
            continue

        rng.shuffle(logs)
        for log in logs:
            key = (log["transactionHash"].lower(), log["logIndex"].lower())
            if key in seen:
                continue
            if not args.skip_wormholescan:
                if not wormholescan_has_tx(args.wormholescan_base, log["transactionHash"]):
                    continue
                time.sleep(args.sleep)
            try:
                bn = int(log["blockNumber"], 16)
                ts = block_timestamp(args.rpc, bn, block_ts_cache)
                rec = build_record(log, ts)
                validate_record(rec)
            except Exception as exc:
                print(f"skipping malformed log {key}: {exc}", file=sys.stderr)
                continue
            records.append(rec)
            seen.add(key)
            print(f"collected {len(records)}/{args.count}: {key[0]} log {int(key[1], 16)}", file=sys.stderr)
            if len(records) >= args.count:
                break

    records.sort(key=lambda r: (int(r["Raw"]["blockNumber"], 16), int(r["Raw"]["logIndex"], 16)))
    with open(args.out, "w", encoding="utf-8") as f:
        json.dump(records[: args.count], f, indent=2)
        f.write("\n")

    if len(records) < args.count:
        print(
            f"WARNING: wrote only {len(records)} records to {args.out}. "
            "Increase --max-attempts, lower --window if the provider rejects ranges, or use --skip-wormholescan.",
            file=sys.stderr,
        )
        return 2

    print(f"wrote {len(records[: args.count])} records to {args.out}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
