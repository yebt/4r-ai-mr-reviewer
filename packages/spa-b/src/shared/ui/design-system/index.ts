// Design-system barrel — atoms + molecules built on the tokens.css /
// DESIGN.md foundation. See DESIGN.md for the atomic-design layering
// convention this follows.

export { default as Button } from './atoms/Button.vue'
export { default as Input } from './atoms/Input.vue'
export { default as Textarea } from './atoms/Textarea.vue'
export { default as Select } from './atoms/Select.vue'
export type { SelectItemOption } from './atoms/Select.vue'
export { default as Checkbox } from './atoms/Checkbox.vue'
export { default as Switch } from './atoms/Switch.vue'
export { default as Text } from './atoms/Text.vue'
export { default as Heading } from './atoms/Heading.vue'
export { default as Icon } from './atoms/Icon.vue'
export { default as Kbd } from './atoms/Kbd.vue'
export { default as Badge } from './atoms/Badge.vue'
export type { BadgeStatus } from './atoms/Badge.vue'
export { default as Spinner } from './atoms/Spinner.vue'
export { default as Skeleton } from './atoms/Skeleton.vue'

export { default as Field } from './molecules/Field.vue'
export { default as ConfirmDialog } from './molecules/ConfirmDialog.vue'
export { default as Alert } from './molecules/Alert.vue'
export { default as Fab } from './molecules/Fab.vue'
export type { AlertStatus } from './molecules/Alert.vue'
