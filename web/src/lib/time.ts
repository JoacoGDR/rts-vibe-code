// nowISO returns the current wall time formatted as the JSON timestamp
// the server expects on commands.
export function nowISO(): string {
  return new Date().toISOString();
}
