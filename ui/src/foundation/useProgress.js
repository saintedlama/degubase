import { ref } from 'vue'

const visible = ref(false)
const pct = ref(0)
const failed = ref(false)

let navPending = 0
let apiPending = 0
let navTimer = null
let apiTimer = null
let growTimer = null

function show() {
  visible.value = true
  failed.value = false
  if (pct.value < 15) pct.value = 15
  grow()
}

function grow() {
  clearTimeout(growTimer)
  if (pct.value < 82) {
    pct.value = pct.value + (82 - pct.value) * 0.07 + 0.3
    growTimer = setTimeout(grow, 350)
  }
}

function finish(ok) {
  clearTimeout(growTimer)
  clearTimeout(navTimer)
  clearTimeout(apiTimer)
  if (!visible.value) return
  if (!ok) failed.value = true
  pct.value = 100
  setTimeout(() => {
    visible.value = false
    pct.value = 0
    failed.value = false
  }, 280)
}

function checkDone(ok) {
  if (navPending === 0 && apiPending === 0) finish(ok)
}

export function navStart() {
  navPending++
  if (!visible.value) {
    clearTimeout(navTimer)
    navTimer = setTimeout(() => {
      if (navPending > 0) show()
    }, 80)
  }
}

export function navDone() {
  navPending = Math.max(0, navPending - 1)
  checkDone(true)
}

export function trackRequest(promise) {
  if (apiPending === 0 && !visible.value) {
    clearTimeout(apiTimer)
    apiTimer = setTimeout(() => {
      if (apiPending > 0) show()
    }, 200)
  }
  apiPending++

  return promise.then(
    (result) => {
      apiPending = Math.max(0, apiPending - 1)
      checkDone(true)
      return result
    },
    (err) => {
      apiPending = Math.max(0, apiPending - 1)
      checkDone(false)
      throw err
    }
  )
}

export function useProgress() {
  return { visible, pct, failed }
}
