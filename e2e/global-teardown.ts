import { writeLcov } from "./coverage";

/**
 * Global teardown — generates lcov coverage report if COVERAGE=1 was set.
 */
export default async function globalTeardown() {
  if (process.env.COVERAGE === "1") {
    writeLcov();
  }
}
