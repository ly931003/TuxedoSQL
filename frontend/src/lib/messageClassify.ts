export type MessageType = 'success' | 'error' | 'info' | (string & {}) | undefined

/**
 * 根据 messageType 返回消息的 CSS 类名。
 *
 * 不再依赖 emoji 前缀匹配 —— messageType 是唯一的分类依据。
 *
 * @param _msg - 消息内容（保留参数，便于未来扩展）
 * @param messageType - 消息类型：success | error | info | undefined
 * @returns CSS 类名：msg-success / msg-error / msg-info
 */
export function getMessageClass(_msg: string, messageType: MessageType): string {
  if (messageType === 'success') return 'msg-success'
  if (messageType === 'error') return 'msg-error'
  return 'msg-info'
}
