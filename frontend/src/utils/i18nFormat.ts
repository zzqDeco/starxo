export type TranslateFn = (key: string, ...args: any[]) => string

export function formatNamedMessage(t: TranslateFn, key: string, named: Record<string, unknown>): string {
  const rendered = t(key, named)
  return replaceNamedPlaceholders(rendered, named)
}

export function replaceNamedPlaceholders(template: string, named: Record<string, unknown>): string {
  return Object.entries(named).reduce((text, [name, value]) => {
    const safeValue = value == null ? '' : String(value)
    return text.replaceAll(`{${name}}`, safeValue)
  }, template)
}
