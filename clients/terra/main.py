import argparse
from terra_sdk.client.lcd import AsyncLCDClient
import asyncio
from terra_sdk.core.wasm import (
    MsgExecuteContract,
)
from terra_sdk.key.mnemonic import MnemonicKey
import base64


def add_default_args(parser):
    parser.add_argument('--rpc', required=True, help='Terra lcd address')
    parser.add_argument('--chain-id', dest="chain_id", required=True, help='Chain ID')
    parser.add_argument('--mnemonic', dest="mnemonic", required=True, help='Mnemonic of the wallet to be used')
    parser.add_argument('--contract', dest="contract", required=True, help='Address of the Wormhole contract')


parser = argparse.ArgumentParser(prog='terra_cli')

subparsers = parser.add_subparsers(help='sub-command help', dest="command")
gov_parser = subparsers.add_parser('execute_governance', help='Execute a governance VAA')
add_default_args(gov_parser)
gov_parser.add_argument('vaa', help='Hex encoded VAA')
post_parser = subparsers.add_parser('post_message', help='Publish a message over the wormhole')
add_default_args(post_parser)
post_parser.add_argument('nonce', help='Nonce of the message', type=int)
post_parser.add_argument('message', help='Hex-encoded message')

gas_prices = {
    "uluna": "0.15",
    "usdr": "0.1018",
    "uusd": "0.15",
    "ukrw": "178.05",
    "umnt": "431.6259",
    "ueur": "0.125",
    "ucny": "0.97",
    "ujpy": "16",
    "ugbp": "0.11",
    "uinr": "11",
    "ucad": "0.19",
    "uchf": "0.13",
    "uaud": "0.19",
    "usgd": "0.2",
}


async def sign_and_broadcast(sequence, deployer, terra, *msgs):
    tx = await deployer.create_and_sign_tx(
        msgs=msgs, fee_denoms=["ukrw", "uusd", "uluna"], sequence=sequence, gas_adjustment=1.4,
    )
    result = await terra.tx.broadcast(tx)
    sequence += 1
    if result.is_tx_error():
        raise RuntimeError(result.raw_log)
    return result


class ContractQuerier:
    def __init__(self, address, terra):
        self.address = address
        self.terra = terra

    def __getattr__(self, item):
        async def result_fxn(**kwargs):
            return await self.terra.wasm.contract_query(self.address, {item: kwargs})

        return result_fxn


class Contract:
    def __init__(self, address, deployer, terra, sequence):
        self.address = address
        self.deployer = deployer
        self.terra = terra
        self.sequence = sequence

    def __getattr__(self, item):
        async def result_fxn(coins=None, **kwargs):
            execute = MsgExecuteContract(
                self.deployer.key.acc_address, self.address, {item: kwargs}, coins=coins
            )
            return await sign_and_broadcast(self.sequence, self.deployer, self.terra, execute)

        return result_fxn

    @property
    def query(self):
        return ContractQuerier(self.address, self.terra)


async def main():
    args = parser.parse_args()
    async with AsyncLCDClient(
            args.rpc, args.chain_id, gas_prices=gas_prices, loop=asyncio.get_event_loop()
    ) as terra:
        deployer = terra.wallet(MnemonicKey(
            mnemonic=args.mnemonic))
        sequence = await deployer.sequence()

        wormhole = Contract(args.contract, deployer, terra, sequence)

        res = dict()
        if args.command == "execute_governance":
            res = await wormhole.submit_v_a_a(vaa=base64.b64encode(bytes.fromhex(args.vaa)).decode("utf-8"))
        elif args.command == "post_message":
            state = await wormhole.query.get_state()
            fee = state["fee"]
            res = await wormhole.post_message(nonce=args.nonce,
                                              message=base64.b64encode(bytes.fromhex(args.message)).decode("utf-8"),
                                              coins={fee["denom"]: fee["amount"]})
        print(res.logs[0].events_by_type["from_contract"])


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(main())
