import { Algodv2, OnApplicationComplete } from "algosdk"
import { Address, ContractAmount, Asset } from "./sdk/AlgorandTypes"
import { Signer } from "./sdk/Signer"
import { Deployer, SignCallback, TealSignCallback } from "./sdk/Deployer"
import { WormholeConfig, WORMHOLE_CONFIG_TESTNET, LOCAL_CONFIG, TestExecutionEnvironmentConfig } from "./sdk/Environment"

// Avoid Jest to print where the console.log messagres are emmited
global.console = require('console');

// We want to actually see the full stacktrace for errors
Error.stackTraceLimit = Infinity;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function improve_errors(task: (() => Promise<void>) | any ) {
    return async () => {
        try {
            await task()
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } catch (error: any) {
            if (error !== undefined) {
                let errorExtraInfo = "No error extra info available."
                if (error.response !== undefined) {
                    errorExtraInfo = error.response.text !== undefined
                        ? JSON.parse(error.response.text).message
                        : error.response
                }
                // We want to preserve the error's stack trace, so we only modify the message
                error.message = error.message + " Error Extra Info: " + errorExtraInfo
                throw error
            }
            throw "UNKNOWN ERROR. Received error variable in catch is undefined"
        }
    }
}

export class SeededRandom {
  constructor(
    private seedA: number = 1234,
    private seedB: number = 4321,
    private seedC: number = 5678,
    private seedD: number = 8765
  ) {}

  public rand(): number {
    // http://pracrand.sourceforge.net/, sfc32
    this.seedA >>>= 0; this.seedB >>>= 0; this.seedC >>>= 0; this.seedD >>>= 0;
    let t = (this.seedA + this.seedB) | 0;
    this.seedA = this.seedB ^ this.seedB >>> 9;
    this.seedB = this.seedC + (this.seedC << 3) | 0;
    this.seedC = (this.seedC << 21 | this.seedC >>> 11);
    this.seedD = this.seedD + 1 | 0;
    t = t + this.seedD | 0;
    this.seedC = this.seedC + t | 0;
    return (t >>> 0) / 4294967296;
  }

  public randRange(lo: number, hi: number): number {
    return this.rand() * (hi - lo) + lo
  }

  public randInt(lo: number, hi: number): number {
    return Math.floor(this.randRange(lo, hi))
  }

  public randBool(): boolean {
    return this.rand() >= 0.5
  }
}

export class TestHelper {
  signCallback: SignCallback
  tealSignCallback: TealSignCallback
  rng: SeededRandom = new SeededRandom()

  public static fromConfig(config: TestExecutionEnvironmentConfig = LOCAL_CONFIG, wormholeConfig: WormholeConfig = WORMHOLE_CONFIG_TESTNET): TestHelper {
    const algoSdk = new Algodv2(config.algod.token, config.algod.server, config.algod.port)
    const deployer = new Deployer(algoSdk, 1_000, undefined, wormholeConfig)
    const signer = new Signer()
    const master = signer.addFromMnemonic(config.masterAccount)
    return new TestHelper(deployer, signer, master)
  }

  constructor(readonly deployer: Deployer, readonly signer: Signer, readonly master: Address) {
    this.signCallback = this.signer.callback
    this.tealSignCallback = this.signer.tealCallback
  }

  public createAccount(): Address {
    return this.signer.createAccount()
  }

  public deployerAccount(): Address {
    return this.master;
  }

  public async waitForTransactionResponse(txId: Address): Promise<Record<string, any>> {
    return this.deployer.waitForTransactionResponse(txId)
  }

  public async fundUser(account: Address, amount: ContractAmount, assetId = 0) {
      const fundTx = assetId === 0 ?
          await this.deployer.makePayTransaction(this.master, account, amount) :
          await this.deployer.makeAssetTransferTransaction(this.master, account, assetId, amount)
      const txId = await this.deployer.signAndSend([fundTx], this.signCallback)
      await this.deployer.waitForTransactionResponse(txId)
      console.log("User " + account + " received " + amount + (assetId == 0 ? " algos" : " assets"))
  }

  public async createAsset(unitName: string, total: ContractAmount | number, decimals = 0): Promise<Asset> {
      const name = "Test Asset " + unitName
      const url = unitName + ".io"
      const assetTx = await this.deployer.makeAssetCreationTransaction(
          this.master, total, decimals, unitName, "Test Asset " + name, name + ".io")
      const creationId = await this.deployer.signAndSend([assetTx], this.signCallback)
      const assetId = (await this.deployer.waitForTransactionResponse(creationId))["asset-index"]
      console.log("Created Test Asset: " + name, assetId)
      return {id: assetId, unitName, name, decimals, url}
  }

  public async optinAsset(account: Address, assetId: number) {
      const optinTx = await this.deployer.makeAssetOptInTransaction(account, assetId)
      const txId = await this.deployer.signAndSend([optinTx], this.signCallback)
      await this.deployer.waitForTransactionResponse(txId)
      console.log("Account " + account + " optin to asset: " + assetId)
  }

  public async optinApp(account: Address, appId: number) {
    const optinTx = await this.deployer.makeCallTransaction(account, appId, OnApplicationComplete.OptInOC, [], [], [], [], "")
    const txId = await this.deployer.signAndSend([optinTx], this.signCallback)
    await this.deployer.waitForTransactionResponse(txId)
    console.log("Account " + account + " optin to app: " + appId)
  }

  public deployAndFund<T extends readonly unknown[], R>(
    f: (deployer: Deployer, deployerAccount: Address, signCallback: SignCallback, ...args: [...T]) => R,
    ...args: T
  ) {
    return f(this.deployer, this.master, this.signCallback, ...args)
  }

  public async clearApps() {
    try {
      await this.deployer.clearApps(this.deployerAccount(), this.signCallback)
      await this.deployer.deleteApps(this.deployerAccount(), this.signCallback)
    } catch (e) {
      console.log(`Clear/delete error being ignored: ${e}`)
    }
  }

  public async createAccountGroup(balances: ContractAmount[][], clearApps = true): Promise<[Address[], Asset[]]> {
    if (clearApps) {
      await this.clearApps()
    }
    // Create accounts
    const accounts = balances.map(() => this.createAccount())

    // Calculate the total asset supply consumed by this request
    const assetSupply = balances.reduce((accum, val) => {
      val.forEach((x, i) => accum[i] = BigInt(accum[i] ?? 0) + BigInt(x))
      return accum
    }, [])

    // Asset 0 is algos, so slice it off and create the remaining assets
    const assets = await Promise.all(assetSupply.slice(1).map((supply, index) => {
      return this.createAsset(`A${index}`, supply)
    }))

    // Give algos and assets to users
    // NOTE: This is not a peformance critical section, so the extra await ticks should be okay
    await Promise.all(balances.map(async (row, i) => {
      // Asset 0 is algos
      await this.fundUser(accounts[i], row[0])

      // Assets 1..n are actual assets that need to be opted in to
      await Promise.all(row.slice(1).flatMap(async (balance, j) => {
        await this.optinAsset(accounts[i], assets[j].id)
        if (balance > 0) {
          await this.fundUser(accounts[i], balance, assets[j].id)
        }
      }))
    }))

    return [accounts, assets]
  }

  public standardAssetGrid(userCount: number, assetCount: number, algoAmount: number | ContractAmount = 2000000, assetAmount: number | ContractAmount = 1000000): ContractAmount[][] {
    return [...Array(userCount)].map(() => [BigInt(algoAmount)].concat([...Array(assetCount)].fill(BigInt(assetAmount))))
  }
}
