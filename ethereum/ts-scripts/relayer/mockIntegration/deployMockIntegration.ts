import {
  init,
  loadChains,
  writeOutputFiles,
  getMockIntegration,
  Deployment,
} from "../helpers/env"
import { deployMockIntegration } from "../helpers/deployments"
import { BigNumber, BigNumberish, BytesLike } from "ethers"
import { tryNativeToHexString, tryNativeToUint8Array } from "@certusone/wormhole-sdk"
import { MockRelayerIntegration__factory } from "../../../sdk/src"
import { wait } from "../helpers/utils"

const processName = "deployMockIntegration"
init()
const chains = loadChains()

async function run() {
  console.log("Start!")
  const output = {
    mockIntegrations: [] as Deployment[],
  }

  for (let i = 0; i < chains.length; i++) {
    const mockIntegration = await deployMockIntegration(chains[i])
    output.mockIntegrations.push(mockIntegration)
  }

  writeOutputFiles(output, processName)

  for (let i = 0; i < chains.length; i++) {
    console.log(`Registering emitters for chainId ${chains[i].chainId}`)
    // note: must use useLastRun = true 
    const mockIntegration = getMockIntegration(chains[i])

    const arg: {
      chainId: BigNumberish
      addr: BytesLike
    }[] = chains.map((c, j) => ({
      chainId: c.chainId,
      addr: "0x" + tryNativeToHexString(output.mockIntegrations[j].address, "ethereum"),
    }))
    await mockIntegration.registerEmitters(arg, { gasLimit: 500000 }).then(wait)
  }
}

run().then(() => console.log("Done!"))
