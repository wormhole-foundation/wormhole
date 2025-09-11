import fs from "fs";
import yargs from "yargs";
import { hideBin } from 'yargs/helpers';
import path from "path";

interface Peer {
  Hostname: string;
  TlsX509: string;
  Port: number;
}

interface Config {
  NumParticipants: number;
  WantedThreshold: number;
  Self: Peer;
  SelfSecret: string;
  StorageLocation: string;
  Peers: Peer[];
}

const createConfigFile = (filePath: string, tlsKeyPath: string, participants: number, threshold: number, selfPeer: string) => {
  if (fs.existsSync(filePath)) {
    console.log(`❌ Config file already exists at ${filePath}`);
    return;
  }

  if (!fs.existsSync(tlsKeyPath)) {
    console.log(`❌ TLS key file not found at ${filePath}`);
    return;
  }

  // TODO: parse signature and verify peer description message
  const {hostname, port, certificate} = JSON.parse(selfPeer);
  const selfPeerObject: Peer = {
    Hostname: hostname,
    TlsX509: certificate,
    Port: port,
  };

  const rawKey = fs.readFileSync(filePath);

  const defaultConfig: Config = {
    NumParticipants: participants,
    WantedThreshold: threshold,
    Self: selfPeerObject,
    SelfSecret: rawKey.toString("base64"),
    StorageLocation: ".",
    Peers: [selfPeerObject],
  };

  fs.writeFileSync(filePath, JSON.stringify(defaultConfig, null, 2), 'utf-8');
  console.log(`✅ Config file created at ${filePath}`);
};

const updateConfigFile = (filePath: string, message: string) => {
  if (!fs.existsSync(filePath)) {
    console.error(`❌ No config file found at ${filePath}`);
    return;
  }

  // TODO: parse signature and verify it
  const {hostname, port, certificate} = JSON.parse(message);

  const rawConfig = fs.readFileSync(filePath, 'utf-8');
  let config: Config;

  try {
    config = JSON.parse(rawConfig);
  } catch (error: any) {
    console.error(`❌ Invalid JSON in config file\n  Error: ${error?.stack || error}`);
    return;
  }

  const peers = config.Peers;
  peers.push({ Hostname: hostname, TlsX509: certificate, Port: port});
  peers.sort((a, b) => lexCompare(a.TlsX509, b.TlsX509));

  fs.writeFileSync(filePath, JSON.stringify(config, null, 2), 'utf-8');
  console.log(`✅ Added peer '${hostname}' to config`);
};

const signCertificate = (certificatePath: string, hostname: string, port: number, guardianKeyPath: string) => {
  if (!fs.existsSync(certificatePath)) {
    console.error(`❌ No certificate file found at ${certificatePath}`);
    return;
  }

  if (!fs.existsSync(guardianKeyPath)) {
    console.error(`❌ No guardian key file found at ${guardianKeyPath}`);
    return;
  }

  // TODO: validate guardian key file format
  // TODO: sign peer description message

  const certificate = fs.readFileSync(certificatePath);

  const message = {
    hostname,
    port,
    certificate: certificate.toString("base64"),
  }

  console.log(`✅ Signed peer description: ${JSON.stringify(message)}`);
};

yargs(hideBin(process.argv))
  .command(
    'create-config --self-peer <own peer message> --participants <number> --threshold <number> --tls-key <path to key file> <path to config file>',
    'Creates a DKG config file',
    (yargs) => {
      return yargs.positional('path', {
        describe: 'Path to create the config file',
        type: 'string',
        demandOption: true,
      })
      .option('self-peer', {
        describe: 'Signed message that describes peer',
        type: 'string',
        demandOption: true,
      })
      .option('participants', {
        describe: 'Total number of guardians participating in TSS protocol',
        type: 'number',
        demandOption: true,
      })
      .option('threshold', {
        describe: 'Number of guardians needed to sign, aka quorum',
        type: 'number',
        demandOption: true,
      })
      .option('tls-key', {
        describe: 'Path to TLS key file',
        type: 'string',
        demandOption: true,
      });
    },
    (argv) => {
      createConfigFile(path.resolve(argv.path), argv.tlsKey, argv.participants, argv.threshold, argv.selfPeer);
    }
  )
  .command(
    'add-peer <path> <message>',
    'Adds a TSS Guardian peer to the configuration',
    (yargs) => {
      return yargs
        .positional('path', {
          describe: 'Path to the config file',
          type: 'string',
          demandOption: true,
        })
        .positional('message', {
          describe: 'Signed message that describes peer',
          type: 'string',
          demandOption: true,
        });
    },
    (argv) => {
      updateConfigFile(path.resolve(argv.path), argv.message);
    }
  )
  .command(
    'sign-certificate --guardian-key <path to key> <path to certificate> <hostname> <port>',
    'Signs the TLS certificate with the v1 guardian key and encodes it for use with DKG and TSS guardian configuration.',
    (yargs) => {
      return yargs.positional('path', {
        describe: 'Path to certificate',
        type: 'string',
        demandOption: true,
      }).positional('hostname', {
        describe: 'Hostname of your peer',
        type: 'string',
        demandOption: true,
      }).positional('port', {
        describe: 'Port of your peer',
        type: 'number',
        demandOption: true,
      }).option("guardian-key", {
        describe: "Path to file with hex encoded, 0x prefixed, guardian key",
        type: 'string',
        demandOption: true,
      });
    },
    (argv) => {
      signCertificate(path.resolve(argv.path), argv.hostname, argv.port, argv.guardianKey);
    }
  )
  .demandCommand()
  .strict()
  .help()
  .parse();



function lexCompare(a: string, b: string): number {
  if (a.length < b.length) return -1;
  if (a.length > b.length) return 1;

  for (let i = 0; i < a.length; ++i) {
    const codeA = a.charCodeAt(i);
    const codeB = b.charCodeAt(i);

    if (codeA < codeB) return -1;
    if (codeA > codeB) return 1;
  }

  return 0;
}