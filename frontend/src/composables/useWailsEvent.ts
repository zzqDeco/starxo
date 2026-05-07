import { onMounted, onUnmounted } from 'vue'
import { EventsOn } from '../../wailsjs/runtime/runtime'

export function useWailsEvent<T = any>(eventName: string, handler: (data: T) => void) {
  let cleanup: (() => void) | undefined

  onMounted(() => {
    cleanup = EventsOn(eventName, handler)
  })

  onUnmounted(() => {
    cleanup?.()
    cleanup = undefined
  })
}
