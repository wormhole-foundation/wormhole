import { Peer } from './types.js';

export class Display {
  private hasProgressBar = false;

  log(message: string): void {
    if (this.hasProgressBar) {
      // Clear the current line and move cursor to beginning
      process.stdout.write('\r\x1b[K');
      console.log(message);
      // The progress bar will be redrawn by the next setProgress call
    } else {
      console.log(message);
    }
  }

  error(message: string, error?: any): void {
    if (this.hasProgressBar) {
      // Clear the current line and move cursor to beginning
      process.stdout.write('\r\x1b[K');
      console.error(message, error || '');
      // The progress bar will be redrawn by the next setProgress call
    } else {
      console.error(message, error || '');
    }
  }

  setProgress(current: number, total: number, label = 'Progress', peers?: Record<string, Peer>): void {
    // Clear the current line and move cursor to beginning
    process.stdout.write('\r\x1b[K');
    
    if (total === 0) {
      process.stdout.write(`${label}: Waiting for guardian data...`);
      this.hasProgressBar = true;
      return;
    }

    // Create progress bar
    const barLength = 40;
    const filledLength = Math.round((current / total) * barLength);
    const emptyLength = barLength - filledLength;
    
    const bar = '█'.repeat(filledLength) + '░'.repeat(emptyLength);
    const percentage = Math.round((current / total) * 100);
    
    // Display progress
    process.stdout.write(
      `${label}: [${bar}] ${current}/${total} guardians (${percentage}%)`
    );

    this.hasProgressBar = true;

    // If complete, finish the line and show completion
    if (current === total && current > 0) {
      process.stdout.write('\n✅ All guardians have submitted their peer data!\n');
      this.hasProgressBar = false;
      
      // Display all peers when complete
      if (peers) {
        this.displayAllPeers(peers);
      }
    }
  }

  private displayAllPeers(peers: Record<string, Peer>): void {
    try {
      this.log('\n📋 All peers are now available:');
      this.log('=====================================');
      
      const peerEntries = Object.entries(peers);
      if (peerEntries.length === 0) {
        this.log('No peers found.');
        return;
      }

      peerEntries.forEach(([guardianAddress, peer], index) => {
        this.log(`${index + 1}. Guardian: ${guardianAddress.slice(0, 10)}...${guardianAddress.slice(-8)}`);
        this.log(`   Hostname: ${peer.Hostname}`);
        this.log(`   Port: ${peer.Port}`);
        this.log(`   TLS Certificate: ${peer.TlsX509.substring(0, 50)}...`);
        this.log('');
      });
      
      this.log(`Total: ${peerEntries.length} peer${peerEntries.length !== 1 ? 's' : ''} collected from guardians`);
      this.log('Guardian submissions complete. Server will continue running...');
    } catch (error) {
      this.error('Error displaying peers:', error);
    }
  }
}
