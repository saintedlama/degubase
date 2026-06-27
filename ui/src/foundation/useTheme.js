import { useDark, useToggle } from '@vueuse/core'

const isDark = useDark()
const toggle = useToggle(isDark)

export function useTheme() {
  return { isDark, toggle }
}
