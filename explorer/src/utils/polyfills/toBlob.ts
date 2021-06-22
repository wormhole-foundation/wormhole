import { isBrowser } from '~/utils/system';

// eslint-disable-next-line @typescript-eslint/unbound-method
if (isBrowser() && !window.HTMLCanvasElement.prototype.toBlob) {
  Object.defineProperty(HTMLCanvasElement.prototype, 'toBlob', {
    value(callback: BlobCallback, type?: string | undefined, quality?: number) {
      setTimeout(() => {
        const binStr = atob(this.toDataURL(type, quality).split(',')[1]);
        const len = binStr.length;
        const arr = new Uint8Array(len);

        for (let i = 0; i < len; i += 1) {
          arr[i] = binStr.charCodeAt(i);
        }

        callback(new Blob([arr], { type: type || 'image/png' }));
      });
    },
  });
}
