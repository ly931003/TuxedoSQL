export interface StructuredTypeHandler {
  detect: (dataType: string) => boolean
  format: (raw: string) => string
  validate: (input: string) => string | null
}

const structuredTypeHandlers: StructuredTypeHandler[] = [
  {
    detect: (dataType: string): boolean => /^(json|jsonb)$/i.test(dataType),
    format: (raw: string): string => {
      try {
        return JSON.stringify(JSON.parse(raw), null, 2)
      } catch {
        return raw
      }
    },
    validate: (input: string): string | null => {
      try {
        JSON.parse(input)
        return null
      } catch (error: unknown) {
        const message = error instanceof Error ? error.message : String(error)
        return `JSON 解析错误: ${message}`
      }
    },
  },
]

/**
 * 提供结构化字段类型的识别、格式化与校验能力。
 */
export function useStructuredCell(): {
  isStructuredType: (dataType: string) => boolean
  formatStructuredValue: (dataType: string, raw: unknown) => unknown
  validateStructuredValue: (dataType: string, input: string) => string | null
} {
  function isStructuredType(dataType: string): boolean {
    return structuredTypeHandlers.some((handler) => handler.detect(dataType))
  }

  function formatStructuredValue(dataType: string, raw: unknown): unknown {
    const handler = structuredTypeHandlers.find((item) => item.detect(dataType))

    if (!handler) {
      return raw
    }

    return handler.format(String(raw))
  }

  function validateStructuredValue(dataType: string, input: string): string | null {
    const handler = structuredTypeHandlers.find((item) => item.detect(dataType))

    if (!handler) {
      return null
    }

    return handler.validate(input)
  }

  return {
    isStructuredType,
    formatStructuredValue,
    validateStructuredValue,
  }
}
