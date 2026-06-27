import { watchEffect } from 'vue'
import { commandSourceId } from '../../api/client.js'

const BASE = import.meta.env.VITE_API_BASE ?? ''

export function useTableEvents(workspaceCodeFn, tableCodeFn, { onCreated, onUpdated, onDeleted } = {}) {
  watchEffect((onCleanup) => {
    const ws = typeof workspaceCodeFn === 'function' ? workspaceCodeFn() : workspaceCodeFn
    const table = typeof tableCodeFn === 'function' ? tableCodeFn() : tableCodeFn
    if (!ws || !table) return

    const es = new EventSource(`${BASE}/api/workspaces/${ws}/tables/${table}/events`, { withCredentials: true })

    es.onmessage = (e) => {
      const ev = JSON.parse(e.data)
      if (ev.command_source_id === commandSourceId) return  // own tab — already applied optimistically
      if (ev.type === 'row.created') onCreated?.(ev.row)
      else if (ev.type === 'row.updated') onUpdated?.(ev.row)
      else if (ev.type === 'row.deleted') onDeleted?.(ev.id)
    }

    onCleanup(() => es.close())
  })
}
