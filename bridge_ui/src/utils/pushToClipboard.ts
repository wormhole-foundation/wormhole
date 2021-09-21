export default function pushToClipboard(content: any) {
  if (!navigator.clipboard) {
    // Clipboard API not available
    return;
  }
  return navigator.clipboard.writeText(content);
}
