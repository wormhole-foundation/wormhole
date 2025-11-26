import yargs from "yargs";
import { getOriginalAsset, getProviderForChain } from "../../chains/generic";
import { RPC_OPTIONS } from "../../consts";
import { getNetwork, chainToChain } from "../../utils";
import { tryUint8ArrayToNative, uint8ArrayToHex } from "../../sdk/array";
import { contracts, toChain } from "@wormhole-foundation/sdk-base";

import { tryHexToNativeStringNear } from "@certusone/wormhole-sdk";

export const command = "origin <chain> <address>";
export const desc = `Print the origin chain and address of the asset that corresponds to the given chain and address.`;
export const builder = (y: typeof yargs) =>
	y
		.positional("chain", {
			describe:
				"Chain that wrapped asset came from. To see a list of supported chains, run `worm chains`",
			type: "string",
			demandOption: true,
		} as const)
		.positional("address", {
			describe: "Address of wrapped asset on origin chain",
			type: "string",
			demandOption: true,
		})
		.option("network", {
			alias: "n",
			describe: "Network of target chain",
			choices: ["mainnet", "testnet", "devnet"],
			default: "mainnet",
			demandOption: false,
		} as const)
		.option("rpc", RPC_OPTIONS);
export const handler = async (
	argv: Awaited<ReturnType<typeof builder>["argv"]>,
) => {
	const consoleWarnTemp = console.warn;
	console.warn = () => {};

	const network = getNetwork(argv.network);

	const res = await getOriginalAsset(
		chainToChain(argv.chain),
		network,
		argv.address,
		argv.rpc,
	);
	const chainName = toChain(res.chainId);

	/**
	 * This is a ridiculous patch for the following issue
	 * worm info origin ethereum 0xb4b9dc1c77bdbb135ea907fd5a08094d98883a35 -n mainnet
	 * Error: uint8ArrayToNative: Use tryHexToNativeStringNear instead.
	 *  at c1d (/Users/kinsyu/Documents/repos/wormhole/clients/js/build/main.js:650:91470)
	 *  at Object.I36 [as handler] (/Users/kinsyu/Documents/repos/wormhole/clients/js/build/main.js:839:40058)
	 *  at process.processTicksAndRejections (node:internal/process/task_queues:105:5)
	 */
	if (chainName === "Near") {
		const h = uint8ArrayToHex(res.assetAddress);
		const provider = await getProviderForChain("Near", network, {
			rpc: argv.rpc,
		});
		const chainName = toChain(res.chainId);

		const tokenBridgeAddress = contracts.tokenBridge.get(network, chainName);
		if (!tokenBridgeAddress) {
			throw Error("Coudln't find Token Bridge address for Near");
		}

		const nearAddress = await tryHexToNativeStringNear(
			provider,
			tokenBridgeAddress,
			h,
		);

		console.log({
			...res,
			assetAddress: nearAddress,
		});
	} else {
		console.log({
			...res,
			assetAddress: tryUint8ArrayToNative(
				res.assetAddress,
				toChain(res.chainId),
			),
		});
	}

	console.warn = consoleWarnTemp;
};
