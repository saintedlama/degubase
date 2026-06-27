import { reactive } from 'vue'

const notifications = reactive([])
let nextId = 0

export function useNotifications() {
  function notify(message, type = 'error', duration = 5000) {
    const id = nextId++
    notifications.push({ id, message, type })
    if (duration > 0) setTimeout(() => dismiss(id), duration)
    return id
  }

  function dismiss(id) {
    const idx = notifications.findIndex(n => n.id === id)
    if (idx !== -1) notifications.splice(idx, 1)
  }

  return { notifications, notify, dismiss }
}
