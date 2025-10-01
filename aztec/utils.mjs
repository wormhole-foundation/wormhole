import { createLogger } from '@aztec/foundation/log';
import { serializePrivateExecutionSteps } from '@aztec/stdlib/kernel';
  
import assert from 'node:assert';
import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
  
  
  const logger = createLogger('bench:profile_capture');
  
  const logLevel = ['silent', 'fatal', 'error', 'warn', 'info', 'verbose', 'debug', 'trace'];
  
  
  const GATE_TYPES = [
    'ecc_op',
    'busread',
    'lookup',
    'pub_inputs',
    'arithmetic',
    'delta_range',
    'elliptic',
    'aux',
    'poseidon2_external',
    'poseidon2_internal',
    'overflow',
  ];
  
  
  export class ProxyLogger {
    static instance;
    logs = [];
  
    constructor() {}
  
    static create() {
      ProxyLogger.instance = new ProxyLogger();
    }
  
    static getInstance() {
      return ProxyLogger.instance;
    }
  
    createLogger(prefix) {
      return new Proxy(createLogger(prefix), {
        get: (target, prop) => {
          if (logLevel.includes(prop)) {
            return function (...data) {
              const loggingFn = prop;
              const args = [loggingFn, prefix, ...data];
              ProxyLogger.getInstance().handleLog(...args);
              target[loggingFn].call(this, ...[data[0], data[1]]);
            };
          } else {
            return target[prop];
          }
        },
      });
    }
  
    handleLog(type, prefix, message, data) {
      this.logs.unshift({ type, prefix, message, data, timestamp: Date.now() });
    }
  
    flushLogs() {
      this.logs = [];
    }
  
    getLogs() {
      return this.logs;
    }
  }
  

  
  function getMinimumTrace(logs) {
    const minimumMessage = 'Minimum required block sizes for structured trace';
    const minimumMessageIndex = logs.findIndex(log => log.message.includes(minimumMessage));
    const candidateLogs = logs.slice(minimumMessageIndex - GATE_TYPES.length, minimumMessageIndex + 5);
  
    const traceLogs = candidateLogs
      .filter(log => GATE_TYPES.some(type => log.message.includes(type)))
      .map(log => log.message.split(/\t|\n/))
      .flat()
      .map(log => log.replace(/\(mem: .*\)/, '').trim())
      .filter(Boolean);
  
    const traceSizes = traceLogs.map(log => {
      const [gateType, gateSizeStr] = log
        .replace(/\n.*\)$/, '')
        .replace(/bb - /, '')
        .split(':')
        .map(s => s.trim());
      const gateSize = parseInt(gateSizeStr);
      assert(GATE_TYPES.includes(gateType), `Gate type ${gateType} is not recognized`);
      return { [gateType]: gateSize };
    });
  
    assert(traceSizes.length === GATE_TYPES.length, 'Decoded trace sizes do not match expected amount of gate types');
    return traceSizes.reduce((acc, curr) => ({ ...acc, ...curr }), {});
  }
  
  function getMaxMemory(logs) {
    const candidateLogs = logs.slice(0, 100).filter(log => /\(mem: .*MiB\)/.test(log.message));
    const usage = candidateLogs.map(log => {
      const memStr = log ? log.message.slice(log.message.indexOf('(mem: ') + 6, log.message.indexOf('MiB') - 3) : '';
      return memStr ? parseInt(memStr) : 0;
    });
    return Math.max(...usage);
  }
  
  export function generateBenchmark(
    flow,
    logs,
    stats,
    privateExecutionSteps,
    proverType,
    error,
  ) {
    let maxMemory = 0;
    let minimumTrace;
    try {
      minimumTrace = getMinimumTrace(logs);
      maxMemory = getMaxMemory(logs);
    } catch {
      logger.warn(`Failed obtain minimum trace and max memory for ${flow}. Did you run with REAL_PROOFS=1?`);
    }
  
    const steps = privateExecutionSteps.reduce((acc, step, i) => {
      const previousAccGateCount = i === 0 ? 0 : acc[i - 1].accGateCount;
      return [
        ...acc,
        {
          functionName: step.functionName,
          gateCount: step.gateCount,
          accGateCount: previousAccGateCount + step.gateCount,
          time: step.timings.witgen,
          oracles: Object.entries(step.timings.oracles ?? {}).reduce(
            (acc, [oracleName, oracleData]) => {
              const total = oracleData.times.reduce((sum, time) => sum + time, 0);
              const calls = oracleData.times.length;
              acc[oracleName] = {
                calls,
                max: Math.max(...oracleData.times),
                min: Math.min(...oracleData.times),
                total,
                avg: total / calls,
              };
              return acc;
            },
            {},
          ),
        },
      ];
    }, []);
    const timings = stats.timings;
    const totalGateCount = steps[steps.length - 1].accGateCount;
    return {
      name: flow,
      timings: {
        total: timings.total,
        sync: timings.sync,
        proving: timings.proving,
        unaccounted: timings.unaccounted,
        witgen: timings.perFunction.reduce((acc, fn) => acc + fn.time, 0),
      },
      rpc: Object.entries(stats.nodeRPCCalls ?? {}).reduce(
        (acc, [RPCName, RPCCalls]) => {
          const total = RPCCalls.times.reduce((sum, time) => sum + time, 0);
          const calls = RPCCalls.times.length;
          acc[RPCName] = {
            calls,
            max: Math.max(...RPCCalls.times),
            min: Math.min(...RPCCalls.times),
            total,
            avg: total / calls,
          };
          return acc;
        },
        {},
      ),
      maxMemory,
      proverType,
      minimumTrace: minimumTrace,
      totalGateCount: totalGateCount,
      steps,
      error,
    };
  }

  export async function captureProfile(
    label,
    interaction,
    opts,
    expectedSteps,
  ) {
    // Make sure the proxy logger starts from a clean slate
    ProxyLogger.getInstance().flushLogs();
    const result = await interaction.profile({ ...opts, profileMode: 'full', skipProofGeneration: true });
    const logs = ProxyLogger.getInstance().getLogs();
    if (expectedSteps !== undefined && result.executionSteps.length !== expectedSteps) {
      throw new Error(`Expected ${expectedSteps} execution steps, got ${result.executionSteps.length}`);
    }
    const benchmark = generateBenchmark(label, logs, result.stats, result.executionSteps, 'wasm', undefined);
  
    const ivcFolder = process.env.CAPTURE_IVC_FOLDER;
    if (ivcFolder) {
      logger.info(`Capturing client ivc execution profile for ${label}`);
  
      const resultsDirectory = join(ivcFolder, label);
      logger.info(`Writing private execution steps to ${resultsDirectory}`);
      await mkdir(resultsDirectory, { recursive: true });
      // Write the client IVC files read by the prover.
      const ivcInputsPath = join(resultsDirectory, 'ivc-inputs.msgpack');
      await writeFile(ivcInputsPath, serializePrivateExecutionSteps(result.executionSteps));
      await writeFile(join(resultsDirectory, 'logs.json'), JSON.stringify(logs, null, 2));
      await writeFile(join(resultsDirectory, 'benchmark.json'), JSON.stringify(benchmark, null, 2));
      logger.info(`Wrote private execution steps to ${resultsDirectory}`);
    }
  
    return result;
  }
  