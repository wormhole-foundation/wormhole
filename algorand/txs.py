from typing import List, Dict, Any, Optional
from base64 import b64decode
from algosdk import transaction
from algosdk.v2client.algod import AlgodClient
from algosdk.v2client.models import SimulateRequest, SimulateRequestTransactionGroup, SimulateTraceConfig

import pprint

class PendingTxnResponse:
    def __init__(self, response: Dict[str, Any]) -> None:
        self.poolError: str = response["pool-error"]
        self.txn: Dict[str, Any] = response["txn"]

        self.applicationIndex: Optional[int] = response.get("application-index")
        self.assetIndex: Optional[int] = response.get("asset-index")
        self.closeRewards: Optional[int] = response.get("close-rewards")
        self.closingAmount: Optional[int] = response.get("closing-amount")
        self.confirmedRound: Optional[int] = response.get("confirmed-round")
        self.globalStateDelta: Optional[Any] = response.get("global-state-delta")
        self.localStateDelta: Optional[Any] = response.get("local-state-delta")
        self.receiverRewards: Optional[int] = response.get("receiver-rewards")
        self.senderRewards: Optional[int] = response.get("sender-rewards")

        self.innerTxns: List[Any] = response.get("inner-txns", [])
        self.logs: List[bytes] = [b64decode(l) for l in response.get("logs", [])]

# TODO: use a more generic type for txs?
def sendTransaction(tx: transaction.ApplicationUpdateTxn, client: AlgodClient):
    txid = tx.get_txid()
    print(f"Sending transaction {txid}")
    client.send_transaction(tx)
    resp = waitForTransaction(client, txid)
    pprint.pprint(resp)
    for x in resp.__dict__["logs"]:
        print(x.hex())

def waitForTransaction(
    client: AlgodClient, txID: str, timeout: int = 10
) -> PendingTxnResponse:
    lastStatus = client.status()
    lastRound = lastStatus["last-round"]
    startRound = lastRound

    while lastRound < startRound + timeout:
        pending_txn = client.pending_transaction_info(txID)

        if pending_txn.get("confirmed-round", 0) > 0:
            return PendingTxnResponse(pending_txn)

        if pending_txn["pool-error"]:
            raise Exception(f"Pool error: {pending_txn['pool-error']}")

        lastStatus = client.status_after_block(lastRound + 1)

        lastRound += 1

    raise Exception(f"Transaction {txID} not confirmed after {timeout} rounds")

def simulateTransaction(txs: List[transaction.ApplicationUpdateTxn], client: AlgodClient):
    txgroup = SimulateRequestTransactionGroup(txns=txs)
    simulateConfig = SimulateTraceConfig(enable=True, stack_change=True, scratch_change=True)
    simulateRequest = SimulateRequest(txn_groups=[txgroup], allow_more_logs=True, exec_trace_config=simulateConfig)
    return client.simulate_transactions(simulateRequest)
