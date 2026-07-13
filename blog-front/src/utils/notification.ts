import { ElNotification, type NotificationHandle, type NotificationOptions, type NotificationType } from 'element-plus'
import 'element-plus/es/components/notification/style/css'
import { isVNode, type VNode } from 'vue'

type NotificationPayload = string | VNode | (Partial<NotificationOptions> & {
  plain?: boolean
})

type AppNotificationType = Exclude<NotificationType, ''>

const defaultTitles: Record<AppNotificationType, string> = {
  primary: '提示',
  success: '操作成功',
  warning: '请注意',
  error: '操作失败',
  info: '提示'
}

const defaultDurations: Record<AppNotificationType, number> = {
  primary: 3600,
  success: 3000,
  warning: 4500,
  error: 5200,
  info: 3800
}

const openNotification = (type: AppNotificationType, payload: NotificationPayload): NotificationHandle => {
  const options: Partial<NotificationOptions> = typeof payload === 'string' || isVNode(payload)
    ? { message: payload }
    : payload

  const { plain: _, customClass, ...rest } = options as Partial<NotificationOptions> & { plain?: boolean }

  return ElNotification({
    title: defaultTitles[type],
    position: 'top-right',
    duration: defaultDurations[type],
    showClose: true,
    offset: 18,
    ...rest,
    type,
    customClass: [
      'dh-notification',
      `dh-notification--${type}`,
      customClass
    ].filter(Boolean).join(' ')
  })
}

export const notify = {
  primary: (payload: NotificationPayload) => openNotification('primary', payload),
  success: (payload: NotificationPayload) => openNotification('success', payload),
  warning: (payload: NotificationPayload) => openNotification('warning', payload),
  error: (payload: NotificationPayload) => openNotification('error', payload),
  info: (payload: NotificationPayload) => openNotification('info', payload),
  closeAll: () => ElNotification.closeAll()
}
