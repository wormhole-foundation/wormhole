import net from 'net';

type PortState = 'open' | 'closed' | 'filtered' | 'error';

interface Peer {
  host: string;
  port: number;
}

interface CheckResult extends Peer {
  ip?: string;
  state: PortState;
  error?: unknown;
  rttNs?: bigint;
}

/** Check single host:port using a full TCP connect */
function checkTcp(
  host: string,
  port: number,
  timeoutMs: number
): Promise<CheckResult> {
  return new Promise<CheckResult>((resolve) => {
    const start = process.hrtime.bigint();
    const socket = new net.Socket();
    let finished = false;

    socket.once('connect', () => {
      if (finished) return;
      finished = true;

      const rttNs = process.hrtime.bigint() - start;
      const ip = socket.remoteAddress;
      resolve({ host, ip, port, state: 'open', rttNs });
      socket.end();
    });

    socket.once('error', (err: NodeJS.ErrnoException) => {
      if (finished) return;
      finished = true;

      const code = err.code;
      resolve({
        host,
        port,
        state: code === "ECONNREFUSED" || code === "ENETUNREACH" || code === "EHOSTUNREACH" || code === "EACCES"
          ? "closed"
          : "error",
        error: code ?? err.message
      });
      socket.destroy();
    });

    socket.setTimeout(timeoutMs, () => {
      if (finished) return;
      finished = true;

      resolve({ host, port, state: 'filtered', error: `timeout(${timeoutMs}ms)` });
      socket.destroy();
    });

    socket.connect({ host, port });
  });
}

async function scanList(
  targets: Peer[],
  concurrency = 100,
  timeoutMs = 3000
): Promise<CheckResult[]> {
  const results: CheckResult[] = [];
  let index = 0;
  const workers: Promise<void>[] = [];

  const worker = async () => {
    while (true) {
      const i = index++;
      if (i >= targets.length) return;

      const target = targets[i];
      try {
        const res = await checkTcp(target.host, target.port, timeoutMs);
        results.push(res);
      } catch (error) {
        results.push({
          host: target.host,
          ip: target.host,
          port: target.port,
          state: 'error',
          error: (error as Error | undefined)?.stack ?? error,
        });
      }
    }
  };

  for (let i = 0; i < Math.min(concurrency, targets.length); i++) {
    workers.push(worker());
  }
  await Promise.all(workers);
  return results;
}

(async () => {
  // yarn run tsx testPort.ts google.com:80 example.org:22 localhost:9999
  // TODO: use yargs, there should be a command to pull the peers from
  // the peer description server.
  // We also need to add options for concurrency and timeout.
  const raw = process.argv.slice(2);
  if (raw.length === 0) {
    console.log('Usage: testPort.ts host:port [host:port] ...');
    process.exit(1);
  }
  const targets = raw.map((s) => {
    const [h, p] = s.split(':');
    return { host: h, port: Number(p) };
  });
  const out = await scanList(targets);
  for (const r of out) {
    // eslint-disable-next-line @typescript-eslint/no-base-to-string, @typescript-eslint/restrict-template-expressions
    console.log(`${r.host}:${r.port} -> ${r.state}${r.rttNs !== undefined ? ` (${r.rttNs}ns)` : ''}${r.error !== undefined ? ` (${r.error})` : ''}`);
  }
})().catch((error: unknown) => {
  console.error((error as Error | undefined)?.stack ?? error);
  process.exit(1);
});
