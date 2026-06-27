import { ref, computed, watch, toValue, onScopeDispose } from 'vue'
import { computePosition, autoUpdate } from '@floating-ui/dom'

export function useFloating(reference, floating, options = {}) {
  const x = ref(0)
  const y = ref(0)
  const isPositioned = ref(false)
  const strategy = options.strategy ?? 'absolute'

  async function update() {
    const refEl = toValue(reference)
    const floatEl = toValue(floating)
    if (!refEl || !floatEl) return
    const { x: nx, y: ny } = await computePosition(refEl, floatEl, {
      placement: toValue(options.placement) ?? 'bottom',
      strategy,
      middleware: toValue(options.middleware) ?? [],
    })
    x.value = nx
    y.value = ny
    isPositioned.value = true
  }

  const floatingStyles = computed(() => ({
    position: strategy,
    left: `${x.value}px`,
    top: `${y.value}px`,
  }))

  let cleanup = null

  watch(
    [() => toValue(reference), () => toValue(floating)],
    ([refEl, floatEl]) => {
      cleanup?.()
      cleanup = null
      if (!refEl || !floatEl) {
        // Reset visibility guard so next open starts hidden until positioned
        isPositioned.value = false
        return
      }
      cleanup = options.whileElementsMounted
        ? options.whileElementsMounted(refEl, floatEl, update)
        : (update(), null)
    },
    // flush:'post' ensures the teleported element is laid out (has dimensions)
    // before computePosition runs, so flip() can correctly detect overflow
    { immediate: true, flush: 'post' }
  )

  watch(
    [() => toValue(options.placement), () => toValue(options.middleware)],
    () => { if (toValue(reference) && toValue(floating)) update() }
  )

  onScopeDispose(() => cleanup?.())

  return { floatingStyles, isPositioned }
}
