import http = require("http");
import client = require("prom-client");

// NOTE:  To create a new metric:
// 1) Create a private counter/gauge with appropriate name and help
// 2) Create a method to set the metric to a value
// 3) Register the metric

export class PromHelper {
  private register = new client.Registry();
  private walletReg = new client.Registry();
  private collectDefaultMetrics = client.collectDefaultMetrics;

  // Actual metrics
  private seqNumGauge = new client.Gauge({
    name: "seqNum",
    help: "Last sent sequence number",
  });
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
  private walletBalance = new client.Gauge({
    name: "wallet_balance",
    help: "The wallet balance",
    labelNames: ["timestamp"],
    registers: [this.walletReg],
  });
  private listenCounter = new client.Counter({
    name: "VAAs_received",
    help: "number of Pyth VAAs received",
  });
  private alreadyExecutedCounter = new client.Counter({
    name: "already_executed",
    help: "number of transfers rejected due to already having been executed",
  });
  private transferTimeoutCounter = new client.Counter({
    name: "transfer_timeout",
    help: "number of transfers that timed out",
  });
  private seqNumMismatchCounter = new client.Counter({
    name: "seq_num_mismatch",
    help: "number of transfers that failed due to sequence number mismatch",
  });
  private retryCounter = new client.Counter({
    name: "retries",
    help: "number of retry attempts",
  });
  private retriesExceededCounter = new client.Counter({
    name: "retries_exceeded",
    help: "number of transfers that failed due to exceeding the retry count",
  });
  private insufficentFundsCounter = new client.Counter({
    name: "insufficient_funds",
    help: "number of transfers that failed due to insufficient funds count",
  });
  // End metrics

  private server = http.createServer(async (req, res) => {
    if (req.url === "/metrics") {
      // Return all metrics in the Prometheus exposition format
      res.setHeader("Content-Type", this.register.contentType);
      res.write(await this.register.metrics());
      res.end(await this.walletReg.metrics());
    }
  });

  constructor(name: string, port: number) {
    this.register.setDefaultLabels({
      app: name,
    });
    this.collectDefaultMetrics({ register: this.register });
    // Register each metric
    this.register.registerMetric(this.seqNumGauge);
    this.register.registerMetric(this.successCounter);
    this.register.registerMetric(this.failureCounter);
    this.register.registerMetric(this.completeTime);
    this.register.registerMetric(this.listenCounter);
    this.register.registerMetric(this.alreadyExecutedCounter);
    this.register.registerMetric(this.transferTimeoutCounter);
    this.register.registerMetric(this.seqNumMismatchCounter);
    this.register.registerMetric(this.retryCounter);
    this.register.registerMetric(this.retriesExceededCounter);
    this.register.registerMetric(this.insufficentFundsCounter);
    // End registering metric

    this.server.listen(port);
  }

  // These are the accessor methods for the metrics
  setSeqNum(sn: number) {
    this.seqNumGauge.set(sn);
  }
  incSuccesses() {
    this.successCounter.inc();
  }
  incFailures() {
    this.failureCounter.inc();
  }
  addCompleteTime(val: number) {
    this.completeTime.observe(val);
  }
  setWalletBalance(bal: number) {
    this.walletReg.clear();
    // this.walletReg = new client.Registry();
    this.walletBalance = new client.Gauge({
      name: "wallet_balance",
      help: "The wallet balance",
      labelNames: ["timestamp"],
      registers: [this.walletReg],
    });
    this.walletReg.registerMetric(this.walletBalance);
    let now = new Date();
    // this.walletDate = now.toString();
    this.walletBalance.set({ timestamp: now.toString() }, bal);
    // this.walletBalance.set(bal);
  }
  incIncoming() {
    this.listenCounter.inc();
  }
  incAlreadyExec() {
    this.alreadyExecutedCounter.inc();
  }
  incTransferTimeout() {
    this.transferTimeoutCounter.inc();
  }
  incSeqNumMismatch() {
    this.seqNumMismatchCounter.inc();
  }
  incRetries() {
    this.retryCounter.inc();
  }
  incRetriesExceeded() {
    this.retriesExceededCounter.inc();
  }
  incInsufficentFunds() {
    this.insufficentFundsCounter.inc();
  }
}
