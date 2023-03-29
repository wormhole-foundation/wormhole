import { ChainInfo, init, loadChains } from "../helpers/env"
import { sendMessage, sleep } from "./messageUtils"
import { Counter, register } from "prom-client"
import Koa from "koa"
import Router from "koa-router"

const promPort = 3000

init()
const chains = loadChains()

export const undeliveredMessages = new Counter({
  name: "undelivered_messages",
  help: "Counter for number of messages that were not delivered",
  labelNames: ["sourceChain", "targetChain"],
})

export const deliveredMessages = new Counter({
  name: "delivered_messages",
  help: "Counter for number of messages that were successfully delivered",
  labelNames: ["sourceChain", "targetChain"],
})

async function run() {
  const chainIntervalIdx = process.argv.findIndex((arg) => arg === "--chainInterval")
  const salvoIntervalIdx = process.argv.findIndex((arg) => arg === "--salvoInterval")
  const chainInterval =
    chainIntervalIdx !== -1 ? Number(process.argv[chainIntervalIdx + 1]) : 5_000
  const salvoInterval =
    salvoIntervalIdx !== -1 ? Number(process.argv[salvoIntervalIdx + 1]) : 60_000

  console.log(`chainInterval: ${chainInterval}`)
  console.log(`salvoInterval: ${salvoInterval}`)

  if (process.argv.find((arg) => arg === "--per-chain")) {
    await perChain(chainInterval, salvoInterval)
  } else {
    await matrix(chainInterval, salvoInterval)
  }
}

async function perChain(chainIntervalMS: number, salvoIntervalMS: number) {
  console.log(`Sending test messages to and from each chain...`)
  for (let salvo = 0; true; salvo++) {
    console.log("")
    console.log(`Sending salvo ${salvo}`)

    for (let i = 0; i < chains.length; ++i) {
      const j = i === 0 ? chains.length - 1 : 0
      await sendMessageAndReportMetric(chains[i], chains[j], chainIntervalMS)
    }

    await sleep(salvoIntervalMS)
  }
}

async function matrix(chainIntervalMS: number, salvoIntervalMS: number) {
  console.log(`Sending test messages to and from every combination of chains...`)
  for (let salvo = 0; true; salvo++) {
    console.log("")
    console.log(`Sending salvo ${salvo}`)

    for (let i = 0; i < chains.length; ++i) {
      for (let j = 0; i < chains.length; ++i) {
        await sendMessageAndReportMetric(chains[i], chains[j], chainIntervalMS)
      }
    }

    await sleep(salvoIntervalMS)
  }
}

async function sendMessageAndReportMetric(
  sourceChain: ChainInfo,
  targetChain: ChainInfo,
  chainInterval: number
) {
  try {
    const notFound = await sendMessage(sourceChain, targetChain, false, true)
    const counter = notFound ? undeliveredMessages : deliveredMessages
    counter
      .labels({
        sourceChain: sourceChain.chainId,
        targetChain: targetChain.chainId,
      })
      .inc()
  } catch (e) {
    console.error(e)
  }
  await sleep(chainInterval)
}

async function launchMetricsServer() {
  const app = new Koa()
  const router = new Router()

  router.get("/metrics", async (ctx, next) => {
    let metrics = await register.metrics()
    ctx.body = metrics
  })

  app.use(router.allowedMethods())
  app.use(router.routes())
  app.listen(promPort, () =>
    console.log(`Prometheus metrics running on port ${promPort}`)
  )
}

console.log("Start!")
run().then(() => console.log("Done!"))
