import { ref, readonly } from 'vue'
import { api } from '../../api/client.js'

const user = ref(null)
const checked = ref(false)
const authDisabled = ref(false)

export function useAuth() {
  async function check() {
    try {
      const setup = await api.getSetupStatus()
      authDisabled.value = setup.auth_disabled === true
      user.value = await api.getMe()
    } catch (e) {
      user.value = null
      // Only enforce the login redirect for real auth failures (401/403).
      // Network errors, 502s, etc. mean we can't determine auth state —
      // treat as permissive so the app doesn't incorrectly send users to /login.
      const isAuthError = e?.status === 401 || e?.status === 403
      if (!isAuthError) {
        authDisabled.value = true
      }
    } finally {
      checked.value = true
    }
  }

  async function login(username, password) {
    user.value = await api.login({ username, password })
  }

  async function logout() {
    if (authDisabled.value) return
    await api.logout()
    user.value = null
  }

  return {
    user: readonly(user),
    checked: readonly(checked),
    authDisabled: readonly(authDisabled),
    check,
    login,
    logout,
  }
}
