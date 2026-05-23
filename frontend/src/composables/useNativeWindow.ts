import {
  WindowCenter,
  WindowGetPosition,
  WindowGetSize,
  WindowIsMaximised,
  WindowSetPosition,
  WindowSetSize,
  WindowToggleMaximise,
  WindowUnmaximise,
} from '../../wailsjs/runtime/runtime'

interface WindowBounds {
  x: number
  y: number
  width: number
  height: number
}

let restoreBounds: WindowBounds | null = null
let toggleInFlight = false

function availableScreenBounds(): WindowBounds {
  const screenInfo = window.screen as Screen & { availLeft?: number; availTop?: number }
  return {
    x: Number(screenInfo.availLeft || 0),
    y: Number(screenInfo.availTop || 0),
    width: Math.max(800, Math.floor(screenInfo.availWidth || window.innerWidth)),
    height: Math.max(600, Math.floor(screenInfo.availHeight || window.innerHeight)),
  }
}

function nearlyEqual(a: number, b: number, tolerance = 8) {
  return Math.abs(a - b) <= tolerance
}

function isZoomed(current: WindowBounds, target: WindowBounds) {
  return nearlyEqual(current.x, target.x)
    && nearlyEqual(current.y, target.y)
    && nearlyEqual(current.width, target.width)
    && nearlyEqual(current.height, target.height)
}

function defaultRestoreBounds(target: WindowBounds): WindowBounds {
  const width = Math.min(1400, Math.max(1000, Math.round(target.width * 0.82)))
  const height = Math.min(900, Math.max(640, Math.round(target.height * 0.82)))
  return {
    x: Math.round(target.x + (target.width - width) / 2),
    y: Math.round(target.y + (target.height - height) / 2),
    width,
    height,
  }
}

function nextFrame() {
  return new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
}

async function fallbackToggleWindowZoom() {
  const [size, position] = await Promise.all([
    WindowGetSize(),
    WindowGetPosition(),
  ])
  const target = availableScreenBounds()
  const current = {
    x: position.x,
    y: position.y,
    width: size.w,
    height: size.h,
  }

  if (isZoomed(current, target)) {
    const restore = restoreBounds || defaultRestoreBounds(target)
    WindowUnmaximise()
    await nextFrame()
    WindowSetPosition(restore.x, restore.y)
    WindowSetSize(restore.width, restore.height)
    if (!restoreBounds) {
      requestAnimationFrame(() => WindowCenter())
    }
    restoreBounds = null
    return
  }

  restoreBounds = current
  WindowUnmaximise()
  await nextFrame()
  WindowSetPosition(target.x, target.y)
  WindowSetSize(target.width, target.height)
}

export async function toggleWindowZoom() {
  if (toggleInFlight) return
  toggleInFlight = true
  try {
    const maximised = await WindowIsMaximised().catch(() => false)
    if (maximised) {
      WindowUnmaximise()
      return
    }
    WindowToggleMaximise()
  } catch {
    await fallbackToggleWindowZoom()
  } finally {
    window.setTimeout(() => {
      toggleInFlight = false
    }, 220)
  }
}
