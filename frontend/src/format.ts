// formatReceivedAt shows the UTC time of an order in the seller's time zone.
// timeZone is for tests; by default the browser's zone is used.
export function formatReceivedAt(iso: string, timeZone?: string): string {
  return new Date(iso).toLocaleString('ru-RU', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZone,
  })
}
