import http = require("http");
import client = require("prom-client");
import { WalletBalance } from "../relayer/walletMonitor";
import { getLogger } from "./logHelper";

// NOTE:  To create a new metric:
// 1) Create a private counter/gauge with appropriate name and help
// 2) Create a method to set the metric to a value
// 3) Register the metric

const logger = getLogger();
export enum PromMode {
  Listen,
  Relay,
  Both,
}

export class PromHelper {
  private _register = new client.Registry();
  private _mode: PromMode;
  private collectDefaultMetrics = client.collectDefaultMetrics;

  // Actual metrics
  private successCounter = new client.Counter({
    name: "successes",
    help: "number of successful relays",
  });
  private failureCounter = new client.Counter({
    name: "failures",
    help: "number of failed relays",
  });
  private completeTime = new client.Histogram({
    name: "complete_time",
    help: "Time is took to complete transfer",
    buckets: [400, 800, 1600, 3200, 6400, 12800],
  });
  private listenCounter = new client.Counter({
    name: "VAAs_received",
    help: "number of VAAs received",
  });
  private alreadyExecutedCounter = new client.Counter({
    name: "already_executed",
    help: "number of transfers rejected due to already having been executed",
  });

  // Wallet metrics
  private walletBalance = new client.Gauge({
    name: "wallet_balance",
    help: "wallet_balance",
    labelNames: ["currency", "chain_id", "wallet", "currency_address"],
  });
  // End metrics

  private server = http.createServer(async (req, res) => {
    // console.log("promHelpers received a request: ", req);
    if (req.url === "/metrics") {
      // Return all metrics in the Prometheus exposition format
      if (this._mode === PromMode.Listen || this._mode == PromMode.Both) {
        res.setHeader("Content-Type", this._register.contentType);
        res.end(await this._register.metrics());
      }
      if (this._mode === PromMode.Relay || this._mode == PromMode.Both) {
        res.setHeader("Content-Type", this._register.contentType);
        res.end(await this._register.metrics());
      }
    }
  });

  constructor(name: string, port: number, mode: PromMode) {
    var mode_name:string = "";
    // Human readable mode name for the metrics
    if (mode === PromMode.Listen) {
      mode_name = "listener";
    } else if (mode === PromMode.Relay) {
      mode_name = "relayer";
    } else if (mode === PromMode.Both) {
      mode_name = "both";
    }

    this._register.setDefaultLabels({
      app: name,
      mode: mode_name,
    });
    this.collectDefaultMetrics({ register: this._register, prefix: "relayer_" });

    this._mode = mode;
    // Register each metric
    if (this._mode === PromMode.Listen || this._mode == PromMode.Both) {
      this._register.registerMetric(this.listenCounter);
    }
    if (this._mode === PromMode.Relay || this._mode == PromMode.Both) {
      this._register.registerMetric(this.successCounter);
      this._register.registerMetric(this.failureCounter);
      this._register.registerMetric(this.alreadyExecutedCounter);
      this._register.registerMetric(this.walletBalance);
    }
    // End registering metric

    this.server.listen(port);
  }

  // These are the accessor methods for the metrics
  incSuccesses() {
    this.successCounter.inc();
  }
  incFailures() {
    this.failureCounter.inc();
  }
  addCompleteTime(val: number) {
    this.completeTime.observe(val);
  }
  incIncoming() {
    this.listenCounter.inc();
  }
  incAlreadyExec() {
    this.alreadyExecutedCounter.inc();
  }

  // Wallet metrics
  handleWalletBalances(balances: WalletBalance[]) {
    logger.debug("Entered handleWalletBalances...");
    // Walk through each wallet
    // create a gauge for the balance
    // set the gauge
    //this.walletMetrics = [];
    for (const bal of balances) {
      try {
        if (bal.currencyName.length === 0) {
          bal.currencyName = "UNK";
        }
        logger.debug(
          "handleWalletBalance - " +
            bal.currencyName +
            " ChainId: " +
            bal.chainId +
            " Wallet: " +
            bal.walletAddress +
            " Mint: " +
            bal.currencyAddressNative +
            " => " +
            bal.balanceFormatted
        );
        let formBal: number;
        if (!bal.balanceFormatted) {
          formBal = 0;
        } else {
          formBal = parseFloat(bal.balanceFormatted);
        }
	this.walletBalance.labels({
	  currency: bal.currencyName,
	  chain_id: bal.chainId,
	  wallet: bal.walletAddress,
	  currency_address: bal.currencyAddressNative,
	}).set(formBal);
      } catch (e: any) {
        // logger.error("handleWalletBalances() - caught error: %o", e);
        if (e.message) {
          logger.error("handleWalletBalances() - caught error: " + e.message);
        } else {
          logger.error("handleWalletBalances() - caught error ");
        }
      }
    }
  }
}
