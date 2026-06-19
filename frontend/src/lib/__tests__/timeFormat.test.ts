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

  it('matches column types case-insensitively', () => {
    expect(formatCellValue('datetime', '2024-01-15T10:30:00Z')).toBe('2024-01-15 10:30:00')
  })

  it('prioritizes DATETIME matching over DATE substring matching', () => {
    expect(formatCellValue('DATETIME', '2024-01-15T00:00:00Z')).toBe('2024-01-15 00:00:00')
  })

  it('returns datetime values without T separator as-is', () => {
    expect(formatCellValue('DATETIME', '2024-01-15 10:30:00')).toBe('2024-01-15 10:30:00')
  })
})
