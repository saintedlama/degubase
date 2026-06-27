// Global setup hook — reserved for future use (e.g. auth token setup).
// Data directory cleanup happens in the Makefile BEFORE Playwright starts
// so that the Go webServer doesn't hold a lock on the database file.
export default async function globalSetup() {}
