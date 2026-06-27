import { trackRequest } from "../foundation/useProgress.js";

const BASE = import.meta.env.VITE_API_BASE ?? "";

// Stable per-tab identity. SSE events carrying this source ID originated from
// this tab and were already applied optimistically — skip them.
export const commandSourceId = "ui-" + crypto.randomUUID();

function request(method, path, body, params) {
  const headers = {};
  if (body) headers["Content-Type"] = "application/json";

  if (method !== "GET") {
    headers["X-Command-Source-Id"] = commandSourceId;
    headers["X-Command-Id"] = crypto.randomUUID();
  }

  let url = `${BASE}${path}`;
  if (params) {
    const qs = new URLSearchParams(params).toString();
    if (qs) url += "?" + qs;
  }

  const promise = fetch(url, {
    method,
    headers,
    credentials: "include",
    body: body ? JSON.stringify(body) : undefined,
  }).then(async (res) => {
    if (res.status === 204) return null;
    const json = await res.json();
    if (!res.ok) {
      const err = new Error(json.error ?? res.statusText);
      err.status = res.status;
      err.code = json.code;
      err.body = json;
      throw err;
    }
    return json;
  });

  return trackRequest(promise);
}

// Like request() but returns { row, revisionId } so callers can track the
// revision ID returned by the server and send it back on the next save to
// bundle rapid edits into a single history entry.
function requestRevision(method, path, body, revisionId) {
  const headers = { "Content-Type": "application/json" };
  headers["X-Command-Source-Id"] = commandSourceId;
  headers["X-Command-Id"] = crypto.randomUUID();
  if (revisionId) headers["X-Revision-Id"] = revisionId;

  const promise = fetch(`${BASE}${path}`, {
    method,
    headers,
    credentials: "include",
    body: JSON.stringify(body),
  }).then(async (res) => {
    const json = await res.json();
    if (!res.ok) {
      const err = new Error(json.error ?? res.statusText);
      err.status = res.status;
      err.code = json.code;
      throw err;
    }
    return { row: json, revisionId: res.headers.get("X-Revision-Id") };
  });

  return trackRequest(promise);
}

export function fileUrl(wsCode, tableCode, rowId, fileId) {
  return `${BASE}/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}/files/${fileId}`;
}

export function thumbnailUrl(wsCode, tableCode, rowId, fileId) {
  return `${BASE}/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}/files/${fileId}/thumbnail`;
}

export const api = {
  // Workspaces
  listWorkspaces: () => request("GET", "/api/workspaces/"),
  createWorkspace: (data) => request("POST", "/api/workspaces/", data),
  getWorkspace: (code) => request("GET", `/api/workspaces/${code}/`),
  updateWorkspace: (code, data) =>
    request("PUT", `/api/workspaces/${code}/`, data),
  deleteWorkspace: (code) => request("DELETE", `/api/workspaces/${code}/`),

  // Tables
  listTables: (wsCode) => request("GET", `/api/workspaces/${wsCode}/tables/`),
  createTable: (wsCode, data) =>
    request("POST", `/api/workspaces/${wsCode}/tables/`, data),
  getTable: (wsCode, tableCode) =>
    request("GET", `/api/workspaces/${wsCode}/tables/${tableCode}/`),
  updateTable: (wsCode, tableCode, data) =>
    request("PUT", `/api/workspaces/${wsCode}/tables/${tableCode}/`, data),
  deleteTable: (wsCode, tableCode) =>
    request("DELETE", `/api/workspaces/${wsCode}/tables/${tableCode}/`),

  // Columns
  listColumns: (wsCode, tableCode) =>
    request("GET", `/api/workspaces/${wsCode}/tables/${tableCode}/columns/`),
  createColumn: (wsCode, tableCode, data) =>
    request(
      "POST",
      `/api/workspaces/${wsCode}/tables/${tableCode}/columns/`,
      data,
    ),
  updateColumn: (wsCode, tableCode, colCode, data) =>
    request(
      "PUT",
      `/api/workspaces/${wsCode}/tables/${tableCode}/columns/${colCode}`,
      data,
    ),
  reorderColumns: (wsCode, tableCode, codes) =>
    request(
      "PUT",
      `/api/workspaces/${wsCode}/tables/${tableCode}/columns/reorder`,
      { codes },
    ),
  deleteColumn: (wsCode, tableCode, colCode) =>
    request(
      "DELETE",
      `/api/workspaces/${wsCode}/tables/${tableCode}/columns/${colCode}`,
    ),

  // Rows
  listRows: (wsCode, tableCode, params = {}) => {
    const qs = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v != null),
    ).toString();
    return request(
      "GET",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${qs ? "?" + qs : ""}`,
    );
  },
  listGroupedRows: (wsCode, tableCode, params = {}) => {
    const qs = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v != null),
    ).toString();
    return request(
      "GET",
      `/api/workspaces/${wsCode}/tables/${tableCode}/groups${qs ? "?" + qs : ""}`,
    );
  },
  getRow: (wsCode, tableCode, rowId) =>
    request(
      "GET",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}`,
    ),
  createRow: (wsCode, tableCode, data) =>
    request(
      "POST",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/`,
      data,
    ),
  updateRow: (wsCode, tableCode, rowId, data, revisionId) =>
    requestRevision(
      "PUT",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}`,
      data,
      revisionId,
    ),
  patchRow: (wsCode, tableCode, rowId, data, revisionId) =>
    requestRevision(
      "PATCH",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}`,
      data,
      revisionId,
    ),
  deleteRow: (wsCode, tableCode, rowId) =>
    request(
      "DELETE",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}`,
    ),
  bulkPatch: (wsCode, tableCode, filters, data, preview = false) =>
    request("POST", `/api/workspaces/${wsCode}/tables/${tableCode}/rows/bulk`, {
      filters,
      data,
      preview,
    }),

  exportCsvUrl: (wsCode, tableCode, params = {}) => {
    const qs = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v != null),
    ).toString();
    return `${BASE}/api/workspaces/${wsCode}/tables/${tableCode}/rows/export.csv${qs ? "?" + qs : ""}`;
  },

  importCsvPreview: async (wsCode, tableCode, file) => {
    const form = new FormData();
    form.append("file", file);
    const res = await fetch(
      `${BASE}/api/workspaces/${wsCode}/tables/${tableCode}/rows/import`,
      {
        method: "POST",
        credentials: "include",
        body: form,
      },
    );
    if (!res.ok) {
      const j = await res.json().catch(() => ({}));
      throw new Error(j.error ?? res.statusText);
    }
    return res.json();
  },

  importCsvConfirm: async (wsCode, tableCode, file, mapping) => {
    const form = new FormData();
    form.append("file", file);
    form.append("mapping", JSON.stringify(mapping));
    const res = await fetch(
      `${BASE}/api/workspaces/${wsCode}/tables/${tableCode}/rows/import`,
      {
        method: "POST",
        credentials: "include",
        body: form,
      },
    );
    if (!res.ok) {
      const j = await res.json().catch(() => ({}));
      throw new Error(j.error ?? res.statusText);
    }
    return res.json();
  },
  listRowHistory: (wsCode, tableCode, rowId) =>
    request(
      "GET",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}/history`,
    ),
  createAnnotation: (wsCode, tableCode, rowId, annotation) =>
    request(
      "POST",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}/history`,
      { annotation },
    ),
  updateAnnotation: (wsCode, tableCode, rowId, histId, annotation) =>
    request(
      "PATCH",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}/history/${histId}`,
      { annotation },
    ),
  deleteAnnotation: (wsCode, tableCode, rowId, histId) =>
    request(
      "DELETE",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}/history/${histId}`,
    ),

  // Referencing records
  listReferencingRows: (wsCode, tableCode, rowId, params = {}) => {
    const qs = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v != null),
    ).toString();
    return request(
      "GET",
      `/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}/referencing${qs ? "?" + qs : ""}`,
    );
  },

  // File upload
  uploadFile: async (wsCode, tableCode, rowId, colCode, file) => {
    const form = new FormData();
    form.append("colCode", colCode);
    form.append("file", file);
    const res = await fetch(
      `${BASE}/api/workspaces/${wsCode}/tables/${tableCode}/rows/${rowId}/files`,
      {
        method: "POST",
        body: form,
      },
    );
    if (!res.ok) {
      const json = await res.json().catch(() => ({}));
      throw new Error(json.error ?? res.statusText);
    }
    return res.json();
  },

  // Auth
  getSetupStatus: () => request("GET", "/api/auth/setup"),
  login: (data) => request("POST", "/api/auth/login", data),
  logout: () => request("POST", "/api/auth/logout"),
  getMe: () => request("GET", "/api/auth/me"),

  // Admin
  listUsers: () => request("GET", "/api/admin/users"),
  createUser: (data) => request("POST", "/api/admin/users", data),
  deleteUser: (id) => request("DELETE", `/api/admin/users/${id}`),

  listSnapshots: () => request("GET", "/api/admin/snapshots"),
  createSnapshot: () => request("POST", "/api/admin/snapshots"),
  restoreSnapshot: (id) =>
    request("POST", `/api/admin/snapshots/${id}/restore`),

  listJobRuns: (params = {}) => request("GET", "/api/admin/jobs", null, params),

  // Workspace tokens
  listTokens: (wsCode) => request("GET", `/api/workspaces/${wsCode}/tokens`),
  createToken: (wsCode, data) =>
    request("POST", `/api/workspaces/${wsCode}/tokens`, data),
  deleteToken: (wsCode, tokenId) =>
    request("DELETE", `/api/workspaces/${wsCode}/tokens/${tokenId}`),

  // Views
  listViews: (wsCode, tableCode) =>
    request("GET", `/api/workspaces/${wsCode}/tables/${tableCode}/views/`),
  createView: (wsCode, tableCode, data) =>
    request(
      "POST",
      `/api/workspaces/${wsCode}/tables/${tableCode}/views/`,
      data,
    ),
  getView: (wsCode, tableCode, viewCode) =>
    request(
      "GET",
      `/api/workspaces/${wsCode}/tables/${tableCode}/views/${viewCode}`,
    ),
  updateView: (wsCode, tableCode, viewCode, data) =>
    request(
      "PUT",
      `/api/workspaces/${wsCode}/tables/${tableCode}/views/${viewCode}`,
      data,
    ),
  deleteView: (wsCode, tableCode, viewCode) =>
    request(
      "DELETE",
      `/api/workspaces/${wsCode}/tables/${tableCode}/views/${viewCode}`,
    ),

  // Scripts
  listScripts: (wsCode) => request("GET", `/api/workspaces/${wsCode}/scripts/`),
  createScript: (wsCode, data) =>
    request("POST", `/api/workspaces/${wsCode}/scripts/`, data),
  getScript: (wsCode, scriptId) =>
    request("GET", `/api/workspaces/${wsCode}/scripts/${scriptId}`),
  updateScript: (wsCode, scriptId, data) =>
    request("PUT", `/api/workspaces/${wsCode}/scripts/${scriptId}`, data),
  deleteScript: (wsCode, scriptId) =>
    request("DELETE", `/api/workspaces/${wsCode}/scripts/${scriptId}`),

  // Script executions
  listScriptExecutions: (wsCode, scriptId, params = {}) =>
    request(
      "GET",
      `/api/workspaces/${wsCode}/scripts/${scriptId}/executions`,
      null,
      params,
    ),
  createScriptExecution: (wsCode, scriptId, data) =>
    request(
      "POST",
      `/api/workspaces/${wsCode}/scripts/${scriptId}/executions`,
      data,
    ),

  // Script env vars
  listScriptEnv: (wsCode) =>
    request("GET", `/api/workspaces/${wsCode}/scripts/env`),
  upsertScriptEnv: (wsCode, key, value, isSecret) =>
    request("POST", `/api/workspaces/${wsCode}/scripts/env`, {
      key,
      value,
      is_secret: isSecret,
    }),
  deleteScriptEnv: (wsCode, envId) =>
    request("DELETE", `/api/workspaces/${wsCode}/scripts/env/${envId}`),

  // Skills
  listSkills: (wsCode) => request("GET", `/api/workspaces/${wsCode}/skills/`),
  createSkill: (wsCode, data) =>
    request("POST", `/api/workspaces/${wsCode}/skills/`, data),
  previewSkill: async (wsCode, data) => {
    const res = await fetch(`${BASE}/api/workspaces/${wsCode}/skills/preview`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error(res.statusText);
    return res.text();
  },
  updateSkill: (wsCode, skillId, data) =>
    request("PATCH", `/api/workspaces/${wsCode}/skills/${skillId}`, data),
  deleteSkill: (wsCode, skillId) =>
    request("DELETE", `/api/workspaces/${wsCode}/skills/${skillId}`),
  skillMdUrl: (wsCode, skillId) =>
    `${BASE}/api/workspaces/${wsCode}/skills/${skillId}/skill.md`,
};
