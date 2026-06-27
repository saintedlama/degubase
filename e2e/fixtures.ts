import { test as base } from "./test";
import { expect } from "@playwright/test";
import type { APIRequestContext } from "@playwright/test";

export type ColSpec = {
  name: string;
  type: string;
  options?: Record<string, unknown>;
};

export interface TableInfo {
  code: string;
  viewCode: string;
  url: string;
}

// ── API helpers ────────────────────────────────────────────────────────────────
// Call these from beforeAll / afterAll or from inside tests.

export async function createWorkspace(
  request: APIRequestContext,
  name = "E2E WS",
): Promise<string> {
  const res = await request.post("/api/workspaces/", { data: { name } });
  expect(res.status()).toBe(201);
  return (await res.json()).code;
}

export async function createTable(
  request: APIRequestContext,
  wsCode: string,
  name: string,
  cols: ColSpec[] = [],
): Promise<TableInfo> {
  const res = await request.post(`/api/workspaces/${wsCode}/tables/`, {
    data: { name },
  });
  expect(res.status()).toBe(201);
  const { code } = await res.json();

  for (const col of cols) {
    const r = await request.post(
      `/api/workspaces/${wsCode}/tables/${code}/columns/`,
      { data: col },
    );
    expect(r.status()).toBe(201);
  }

  const views: Array<{ type: string; code: string }> = await (
    await request.get(`/api/workspaces/${wsCode}/tables/${code}/views/`)
  ).json();
  const viewCode = views.find((v) => v.type === "tabular")?.code ?? "";

  return {
    code,
    viewCode,
    url: `/workspaces/${wsCode}/tables/${code}/views/${viewCode}`,
  };
}

export async function seedRow(
  request: APIRequestContext,
  wsCode: string,
  tableCode: string,
  data: Record<string, unknown> = {},
): Promise<number> {
  const res = await request.post(
    `/api/workspaces/${wsCode}/tables/${tableCode}/rows/`,
    {
      data: { data },
    },
  );
  expect(res.status()).toBe(201);
  return (await res.json()).id;
}

export async function cleanup(
  request: APIRequestContext,
  wsCode: string,
): Promise<void> {
  if (wsCode) await request.delete(`/api/workspaces/${wsCode}`).catch(() => {});
}

// ── Fixture ────────────────────────────────────────────────────────────────────
// Import `test` from here when each test needs its own isolated workspace
// (per-test scope, automatic cleanup). The `page` fixture inherits automatic
// coverage collection from ./test when COVERAGE=1 is set.

export const test = base.extend<{ wsCode: string }>({
  wsCode: async ({ request }, use) => {
    const code = await createWorkspace(request);
    await use(code);
    await cleanup(request, code);
  },
});

export { expect };
