from algosdk.v2client.algod import AlgodClient
from algosdk import account, mnemonic, transaction
import os
from teal import hashProgram
from txs import sendTransaction, simulateTransaction
import argparse

import pprint


# approve and clear must point to the respective bytecode binaries
def buildUpgradeTx(approval_file: str, clear_file: str, contract_id, network, suggested_params) -> transaction.ApplicationUpdateTxn:
    print("Upgrading the core contracts")
    if approval_file == "" or clear_file == "":
        raise ValueError("Missing approve and clear programs")
    else:
        pprint.pprint([approval_file, clear_file])
        with open(approval_file, "rb") as f:
            approval_program = f.read()
            # pprint.pprint(approval_program)
        with open(clear_file, "rb") as f:
            clear_program = f.read()
            # pprint.pprint(clear_program)

    pprint.pprint(f"Approval program hash: {hashProgram(approval_program)}")

    txn = transaction.ApplicationUpdateTxn(
        index=contract_id,
        sender=network.sender_address,
        approval_program=approval_program,
        clear_program=clear_program,
        app_args=[],
        sp=suggested_params,
    )

    # Not strictly necessary for simulations 🤷
    return txn.sign(network.private_key)


class Testnet():
    algod_url = "https://node.testnet.algoexplorerapi.io"
    algod_token = ""
    indexer_url = "https://indexer.testnet.algoexplorerapi.io"
    coreid = 86525623
    tokenid = 86525641

    def __init__(self):
        self.private_key = mnemonic.to_private_key(os.environ["ALGORAND_KEY_TESTNET"])
        self.sender_address = account.address_from_private_key(self.private_key)

class Mainnet():
    algod_url = "https://node.algoexplorerapi.io"
    algod_token = ""
    indexer_url = "https://indexer.algoexplorerapi.io"
    coreid = 842125965
    tokenid = 842126029

    def __init__(self):
        self.private_key = mnemonic.to_private_key(os.environ["ALGORAND_KEY"])
        self.sender_address = account.address_from_private_key(self.private_key)


def create_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description='Builds Algorand contract upgrade tx and submits it')

    parser.add_argument('--network', type=str, choices=["mainnet", "testnet"], help='Network in which to perform the upgrade', default="testnet")

    subcommands = parser.add_subparsers(dest="contract", required=True)
    core = subcommands.add_parser("core", help="Selects Core for upgrade")
    token_bridge = subcommands.add_parser("token-bridge", help="Selects Token Bridge for upgrade")
    core_subcommands = core.add_subparsers(dest="subcommand", required=True)
    token_bridge_subcommands = token_bridge.add_subparsers(dest="subcommand", required=True)

    core.add_argument('--approve', type=str, help='Core approve contract binary file', default="artifacts/core_approve.teal.bin")
    core.add_argument('--clear', type=str, help='Core clear contract binary file', default="artifacts/core_clear.teal.bin")

    token_bridge.add_argument('--approve', type=str, help='Token Bridge approve contract binary file', default="artifacts/token_approve.teal.bin")
    token_bridge.add_argument('--clear', type=str, help='Token Bridge clear contract binary file', default="artifacts/token_clear.teal.bin")

    core_subcommands.add_parser("simulate", help="Simulates execution of Algorand core contract upgrade")
    token_bridge_subcommands.add_parser("simulate", help="Simulates execution of Algorand token bridge contract upgrade")
    core_subcommands.add_parser("execute", help="Submits tx to upgrade Algorand core contract")
    token_bridge_subcommands.add_parser("execute", help="Submits tx to upgrade Algorand token bridge contract")

    return parser

if __name__ == "__main__":
    parser = create_parser()
    args = parser.parse_args()

    if args.network == "testnet":
        network = Testnet()
    else:
        network = Mainnet()

    if args.contract == "core":
        contract_id = network.coreid
    elif args.contract == "token-bridge":
        contract_id = network.tokenid

    client = AlgodClient(network.algod_token, network.algod_url)
    tx = buildUpgradeTx(args.approve, args.clear, contract_id, network, client.suggested_params())
    if args.subcommand == "execute":
        sendTransaction(tx, client)
    elif args.subcommand == "simulate":
        result = simulateTransaction([tx], client)
        print(f"Simulation: {result}")
    print("Complete")
