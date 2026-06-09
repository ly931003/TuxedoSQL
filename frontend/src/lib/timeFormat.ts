/**
 * 根据 MySQL 列类型格式化单元格值，支持时间相关类型的智能显示。
 *
 * 时间值在 Go 侧经 `parseTime=true` + `loc=` DSN 参数处理后，
 * 以 RFC 3339 字符串（如 "2024-01-15T10:30:00+08:00" 或 "2024-01-15T10:30:00Z"）
 * 传输到前端，本模块负责将其转为人类可读格式。
 */

/**
 * 格式化单个单元格的值。
 *
 * @param colType - MySQL 列类型字符串（如 "DATETIME", "TIMESTAMP", "DATE", "TIME"），
 *   来自 Go `ct.DatabaseTypeName()`。
 * @param value - 单元格值，通常是从 JSON 反序列化后的字符串或原始类型。
 * @returns 格式化后的显示字符串。null/undefined 返回空串。
 */
export function formatCellValue(colType: string, value: unknown): string {
  if (value === null || value === undefined) {
    return ''
  }

  if (typeof value !== 'string') {
    return String(value)
  }

  // 类型匹配（大小写不敏感），注意排除子串冲突：
  // DATETIME 和 TIMESTAMP 必须优先于 DATE / TIME 匹配
  if (/^DATETIME/i.test(colType) || /^TIMESTAMP/i.test(colType)) {
    return formatDatetime(value)
  }
  if (/^DATE/i.test(colType)) {
    return formatDate(value)
  }
  if (/^TIME/i.test(colType)) {
    return formatTime(value)
  }

  return String(value)
}

/**
 * 格式化 DATETIME / TIMESTAMP 值。
 *
 * RFC 3339 输入: "2024-01-15T10:30:00+08:00" 或 "2024-01-15T10:30:00Z"
 * 输出:        "2024-01-15 10:30:00"
 *
 * 时区已在 Go 侧通过 DSN loc 参数转换，前端去掉 T 分隔符和偏移量即可。
 */
function formatDatetime(rfc3339: string): string {
  const tIdx = rfc3339.indexOf('T')
  if (tIdx === -1) return rfc3339

  const datePart = rfc3339.slice(0, tIdx)
  const afterT = rfc3339.slice(tIdx + 1)
  const timePart = afterT.replace(/[+-]\d{2}:\d{2}$|Z$/, '')

  return `${datePart} ${timePart}`
}

/**
 * 格式化 DATE 值。
 *
 * RFC 3339 输入: "2024-01-15T00:00:00Z" 或 "2024-01-15"
 * 输出:        "2024-01-15"
 */
function formatDate(rfc3339: string): string {
  if (rfc3339.includes('T')) {
    return rfc3339.slice(0, rfc3339.indexOf('T'))
  }
  return rfc3339
}

/**
 * 格式化 TIME 值。
 *
 * Go MySQL driver 将 TIME 列编码为带有虚拟日期（0000-01-01）的 time.Time。
 * RFC 3339 输入: "0000-01-01T10:30:00Z"
 * 输出:        "10:30:00"
 */
function formatTime(rfc3339: string): string {
  const tIdx = rfc3339.indexOf('T')
  if (tIdx === -1) return rfc3339

  const afterT = rfc3339.slice(tIdx + 1)
  // 去掉尾部的 "+08:00"、"-05:00" 或 "Z"
  return afterT.replace(/[+-]\d{2}:\d{2}$|Z$/, '')
}
