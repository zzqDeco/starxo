import type { SSHConfig } from '@/types/config'

export function formatSSHError(error: unknown, ssh?: Partial<SSHConfig>): string {
  const message = errorMessage(error)
  if (!shouldShowMacOSLocalNetworkHint(message, ssh?.host)) {
    return message
  }
  return message
}

export function shouldShowMacOSLocalNetworkHint(message: string, host?: string): boolean {
  return isMacPlatform() && isPermissionLikeLocalNetworkError(message) && isLocalNetworkHost(host)
}

export function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message
  if (typeof error === 'string') return error
  if (error && typeof error === 'object' && 'message' in error) {
    const value = (error as { message?: unknown }).message
    if (typeof value === 'string' && value.trim()) return value
  }
  return String(error)
}

function isMacPlatform(): boolean {
  const nav = globalThis.navigator as (Navigator & { userAgentData?: { platform?: string } }) | undefined
  const platform = `${nav?.userAgentData?.platform || nav?.platform || ''}`.toLowerCase()
  const userAgent = `${nav?.userAgent || ''}`.toLowerCase()
  return platform.includes('mac') || userAgent.includes('mac os x')
}

function isPermissionLikeLocalNetworkError(message: string): boolean {
  return /\bno route to host\b/i.test(message) ||
    /\bnetwork is unreachable\b/i.test(message) ||
    /\boperation not permitted\b/i.test(message) ||
    /\bpermission denied\b/i.test(message)
}

function isLocalNetworkHost(host?: string): boolean {
  const normalized = normalizeHost(host)
  if (!normalized) return false

  if (
    normalized === 'localhost' ||
    normalized.endsWith('.localhost') ||
    normalized.endsWith('.local') ||
    normalized.endsWith('.lan')
  ) {
    return true
  }

  if (isPrivateIPv4(normalized)) return true
  if (isPrivateIPv6(normalized)) return true

  return false
}

function normalizeHost(host?: string): string {
  const value = `${host || ''}`.trim().toLowerCase()
  if (!value) return ''
  if (value.startsWith('[')) {
    const end = value.indexOf(']')
    if (end > 0) return value.slice(1, end)
  }
  const colonCount = (value.match(/:/g) || []).length
  if (colonCount === 1 && /:\d+$/.test(value)) {
    return value.replace(/:\d+$/, '')
  }
  return value
}

function isPrivateIPv4(host: string): boolean {
  const parts = host.split('.')
  if (parts.length !== 4) return false
  const octets = parts.map((part) => {
    if (!/^\d+$/.test(part)) return Number.NaN
    return Number(part)
  })
  if (octets.some((octet) => !Number.isInteger(octet) || octet < 0 || octet > 255)) return false

  const [a, b] = octets
  return (
    a === 0 ||
    a === 10 ||
    a === 127 ||
    (a === 100 && b >= 64 && b <= 127) ||
    (a === 169 && b === 254) ||
    (a === 172 && b >= 16 && b <= 31) ||
    (a === 192 && b === 168) ||
    (a >= 224 && a <= 239)
  )
}

function isPrivateIPv6(host: string): boolean {
  const value = host.split('%', 1)[0]
  return (
    value === '::' ||
    value === '::1' ||
    value.startsWith('fe80:') ||
    value.startsWith('fc') ||
    value.startsWith('fd') ||
    value.startsWith('ff')
  )
}
