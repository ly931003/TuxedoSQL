import { getMessageClass } from '../messageClassify'

describe('getMessageClass', () => {
  it('returns msg-success for success type', () => {
    expect(getMessageClass('any msg', 'success')).toBe('msg-success')
  })

  it('returns msg-error for error type', () => {
    expect(getMessageClass('any msg', 'error')).toBe('msg-error')
  })

  it('returns msg-info for info type', () => {
    expect(getMessageClass('any msg', 'info')).toBe('msg-info')
  })

  it('returns msg-info for undefined type', () => {
    expect(getMessageClass('any msg', undefined)).toBe('msg-info')
  })

  it('returns msg-info for empty string type', () => {
    expect(getMessageClass('any msg', '')).toBe('msg-info')
  })

  it('ignores emoji prefix when no type is given — emoji fallback removed', () => {
    expect(getMessageClass('✅ audit message', undefined)).toBe('msg-info')
  })

  it('type parameter takes priority over emoji prefix', () => {
    expect(getMessageClass('✅ audit message', 'success')).toBe('msg-success')
  })
})
