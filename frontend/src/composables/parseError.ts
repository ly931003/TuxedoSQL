/**
 * 将未知错误对象解析为人类可读的字符串。
 * 支持 Error 实例、带 message 属性的对象、以及 JSON 格式的错误。
 */
export function parseError(err: unknown): string {
  if (err instanceof Error) {
    try {
      const p = JSON.parse(err.message)
      if (p?.message) return String(p.message)
    } catch {
      /* JSON 解析失败，使用原始 message */
    }
    return err.message
  }
  if (err && typeof err === 'object') {
    const msg = (err as Record<string, unknown>).message
    if (typeof msg === 'string') return msg
  }
  const raw = String(err)
  try {
    const p = JSON.parse(raw)
    if (p?.message) return String(p.message)
  } catch {
    /* 非 JSON 字符串 */
  }
  return raw
}
