/**
 * Coverage collector for Playwright e2e tests instrumented by vite-plugin-istanbul.
 *
 * When COVERAGE=1:
 *   1. vite-plugin-istanbul instruments source code.
 *   2. Call collectCoverage(page) after page interactions to read
 *      window.__coverage__ and flush it to coverage/raw/<pid>.json.
 *   3. global-teardown.ts calls writeLcov() which merges all raw files
 *      and produces coverage/lcov.info.
 */
import type { Page } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";

// ── Paths ──────────────────────────────────────────────────────────────────

const RAW_DIR = path.resolve(__dirname, "..", "coverage", "raw");

// ── Collect ────────────────────────────────────────────────────────────────

export async function collectCoverage(page: Page) {
  if (process.env.COVERAGE !== "1") return;

  try {
    const cov: Record<string, any> | undefined = await page.evaluate(
      () => (window as any).__coverage__,
    );
    if (!cov || Object.keys(cov).length === 0) return;

    // Filter to project sources only
    const filtered: Record<string, any> = {};
    for (const [k, v] of Object.entries(cov)) {
      if (k.includes("node_modules") || k.includes("vite/")) continue;
      filtered[k] = v;
    }
    if (Object.keys(filtered).length === 0) return;

    // Write to a per-worker file (global teardown runs in a separate process)
    fs.mkdirSync(RAW_DIR, { recursive: true });
    const outPath = path.join(RAW_DIR, `worker-${process.pid}.json`);
    fs.writeFileSync(outPath, JSON.stringify(filtered));

    console.log(
      `[coverage] flushed ${Object.keys(filtered).length} entries → ${outPath}`,
    );
  } catch {
    // Coverage collection is best-effort — never fail a test over it
  }
}

// ── Write lcov ─────────────────────────────────────────────────────────────

export function writeLcov(
  outputDir = path.resolve(__dirname, "..", "coverage"),
) {
  const rawFiles = fs.existsSync(RAW_DIR)
    ? fs.readdirSync(RAW_DIR).filter((f) => f.endsWith(".json"))
    : [];

  if (rawFiles.length === 0) {
    console.warn(
      "[coverage] No raw coverage files found — did you set COVERAGE=1?",
    );
    return;
  }

  const merged: Record<string, any> = {};
  for (const file of rawFiles) {
    const data = JSON.parse(fs.readFileSync(path.join(RAW_DIR, file), "utf-8"));
    for (const [fileKey, fileCov] of Object.entries(data)) {
      if (merged[fileKey]) {
        const existing = merged[fileKey];
        existing.s = mergeHits(existing.s, fileCov.s);
        existing.b = mergeHits(existing.b, fileCov.b);
        existing.f = mergeHits(existing.f, fileCov.f);
      } else {
        merged[fileKey] = fileCov;
      }
    }
  }

  const libCoverage = require("istanbul-lib-coverage");
  const libReport = require("istanbul-lib-report");
  const reports = require("istanbul-reports");

  const map = libCoverage.createCoverageMap(merged);

  const projectRoot = path.resolve(__dirname, "..", "ui");
  const filteredMap = libCoverage.createCoverageMap();
  for (const file of map.files()) {
    if (file.includes("node_modules") || file.includes("vite/")) continue;
    const fc = map.fileCoverageFor(file);
    filteredMap.addFileCoverage(fc);
  }

  const context = libReport.createContext({
    dir: outputDir,
    coverageMap: filteredMap,
  });
  const lcov = reports.create("lcovonly", { projectRoot });
  lcov.execute(context);

  // Print text summary to stdout for CI logs
  const textSummary = reports.create("text-summary");
  textSummary.execute(context);

  // Clean up raw files
  for (const file of rawFiles) {
    fs.unlinkSync(path.join(RAW_DIR, file));
  }

  console.log(
    `[coverage] lcov written to ${outputDir}/lcov.info (${filteredMap.files().length} files)`,
  );
}

function mergeHits(
  existing: Record<string, number> | undefined,
  incoming: Record<string, number> | undefined,
): Record<string, number> {
  if (!existing) return { ...(incoming ?? {}) };
  if (!incoming) return existing;
  const out = { ...existing };
  for (const [k, v] of Object.entries(incoming)) {
    out[k] = (out[k] ?? 0) + v;
  }
  return out;
}
