import { onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

export function useWailsEvent<T = any>(eventName: string, handler: (data: T) => void) {
  onMounted(() => {
    EventsOn(eventName, handler)
  })
  onUnmounted(() => {
    EventsOff(eventName)
  })
}
