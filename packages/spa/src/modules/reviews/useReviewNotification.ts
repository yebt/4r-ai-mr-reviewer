import { ref } from 'vue'
import { useLocalStorage } from '@vueuse/core'

// Desktop-notification helper for the review detail page: the user opts in (which
// requests the browser permission), and when a watched review finishes we fire a
// Notification — but only while the tab is hidden, so a focused user gets an
// in-app toast from the caller instead of a redundant desktop popup.
export function useReviewNotification() {
  const supported = typeof window !== 'undefined' && 'Notification' in window
  const permission = ref<NotificationPermission>(supported ? Notification.permission : 'denied')
  // Opt-in preference, remembered across reloads. The browser permission is the
  // real gate; this just records that the user asked to be notified.
  const enabled = useLocalStorage('reviews:notifyOnDone', false)

  // enable requests permission (if not decided yet) and records the opt-in.
  async function enable() {
    if (!supported) return
    permission.value =
      Notification.permission === 'default'
        ? await Notification.requestPermission()
        : Notification.permission
    enabled.value = permission.value === 'granted'
  }

  // notify fires a desktop notification when the tab is hidden and the user has
  // opted in with permission granted. Returns true if a notification was shown,
  // so the caller can fall back to a toast when it wasn't (e.g. tab focused).
  function notify(title: string, body: string): boolean {
    if (!supported || !enabled.value || permission.value !== 'granted') return false
    if (document.visibilityState !== 'hidden') return false
    const n = new Notification(title, { body })
    n.onclick = () => {
      window.focus()
      n.close()
    }
    return true
  }

  return { supported, permission, enabled, enable, notify }
}
