import { formatCellValue } from '../timeFormat'

describe('formatCellValue', () => {
  it('returns empty string for null', () => {
    expect(formatCellValue('DATETIME', null)).toBe('')
  })

  it('returns empty string for undefined', () => {
    expect(formatCellValue('DATETIME', undefined)).toBe('')
  })

  it('stringifies non-string values', () => {
    expect(formatCellValue('INT', 42)).toBe('42')
  })

  it('formats RFC 3339 datetime with offset', () => {
    expect(formatCellValue('DATETIME', '2024-01-15T10:30:00+08:00')).toBe('2024-01-15 10:30:00')
  })

  it('formats RFC 3339 datetime with Z suffix', () => {
    expect(formatCellValue('DATETIME', '2024-01-15T10:30:00Z')).toBe('2024-01-15 10:30:00')
  })

  it('formats timestamp values like datetime values', () => {
    expect(formatCellValue('TIMESTAMP', '2024-01-15T10:30:00Z')).toBe('2024-01-15 10:30:00')
  })

  it('formats RFC 3339 date values to date only', () => {
    expect(formatCellValue('DATE', '2024-01-15T00:00:00Z')).toBe('2024-01-15')
  })

  it('keeps plain date values unchanged', () => {
    expect(formatCellValue('DATE', '2024-01-15')).toBe('2024-01-15')
  })

  it('formats time values from RFC 3339 strings', () => {
    expect(formatCellValue('TIME', '0000-01-01T10:30:00Z')).toBe('10:30:00')
  })

  it('returns non-time strings unchanged for other types', () => {
    expect(formatCellValue('VARCHAR', 'hello')).toBe('hello')
  })

  it('pretty-prints valid JSON strings for JSON columns', () => {
    expect(formatCellValue('JSON', '{"foo":"bar","count":1}')).toBe(`{
  "foo": "bar",
  "count": 1
}`)
  })

  it('returns invalid JSON strings unchanged for JSON columns', () => {
    expect(formatCellValue('JSON', '{foo:bar}')).toBe('{foo:bar}')
  })

  it('pretty-prints valid JSON strings for JSONB columns', () => {
    expect(formatCellValue('JSONB', '{"items":[1,2]}')).toBe(`{
  "items": [
    1,
    2
  ]
}`)
  })

  it('does not format JSON-like strings for non-JSON columns', () => {
    expect(formatCellValue('VARCHAR', '{"foo":"bar"}')).toBe('{"foo":"bar"}')
  })

  it('matches column types case-insensitively', () => {
    expect(formatCellValue('datetime', '2024-01-15T10:30:00Z')).toBe('2024-01-15 10:30:00')
  })

  it('prioritizes DATETIME matching over DATE substring matching', () => {
    expect(formatCellValue('DATETIME', '2024-01-15T00:00:00Z')).toBe('2024-01-15 00:00:00')
  })

  it('returns datetime values without T separator as-is', () => {
    expect(formatCellValue('DATETIME', '2024-01-15 10:30:00')).toBe('2024-01-15 10:30:00')
  })

  it('returns empty string for null JSON values', () => {
    expect(formatCellValue('JSON', null)).toBe('')
  })

  it('handles empty string for JSON columns', () => {
    expect(formatCellValue('JSON', '')).toBe('')
  })

  it('keeps whitespace-only JSON input unchanged', () => {
    expect(formatCellValue('JSON', '  ')).toBe('  ')
  })

  it('passes through JSON scalar values', () => {
    expect(formatCellValue('JSON', 'null')).toBe('null')
    expect(formatCellValue('JSON', 'true')).toBe('true')
    expect(formatCellValue('JSON', '42')).toBe('42')
  })
})
