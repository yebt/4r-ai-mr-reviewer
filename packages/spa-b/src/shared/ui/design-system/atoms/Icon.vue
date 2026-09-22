<script setup lang="ts">
/**
 * Thin wrapper around lucide-vue-next. One atom, one job: render an icon at
 * a consistent size step from a simple kebab-case `name` (e.g. `settings`,
 * `git-branch`) per DESIGN.md — no `lucide:` prefix.
 *
 * Only the icons actually used by the app are imported below (named imports
 * are tree-shaken by the bundler, so unused icons never ship). An unknown
 * `name` renders nothing rather than throwing.
 */
import { computed, type Component } from 'vue'
import {
  Activity,
  Cable,
  Check,
  ChevronDown,
  CircleCheck,
  CircleX,
  Ellipsis,
  GitBranch,
  GitPullRequest,
  Hammer,
  House,
  Info,
  Layers,
  List,
  LogOut,
  Minus,
  Monitor,
  Moon,
  PlugZap,
  Plus,
  Search,
  Send,
  Settings,
  Settings2,
  Shield,
  Sun,
  TriangleAlert,
  User,
  Users,
  X,
} from 'lucide-vue-next'

/** kebab-case icon name -> lucide-vue-next component. Add new icons here as
 * they're used elsewhere in the app. */
const icons: Record<string, Component> = {
  activity: Activity,
  cable: Cable,
  check: Check,
  'chevron-down': ChevronDown,
  'circle-check': CircleCheck,
  'circle-x': CircleX,
  ellipsis: Ellipsis,
  'git-branch': GitBranch,
  'git-pull-request': GitPullRequest,
  hammer: Hammer,
  house: House,
  info: Info,
  layers: Layers,
  list: List,
  'log-out': LogOut,
  minus: Minus,
  monitor: Monitor,
  moon: Moon,
  'plug-zap': PlugZap,
  plus: Plus,
  search: Search,
  send: Send,
  settings: Settings,
  'settings-2': Settings2,
  shield: Shield,
  sun: Sun,
  'triangle-alert': TriangleAlert,
  user: User,
  users: Users,
  x: X,
}

const sizeMap = {
  xs: 'size-3',
  sm: 'size-3.5',
  md: 'size-4',
  lg: 'size-5',
  xl: 'size-6',
} as const

type IconSize = keyof typeof sizeMap

const props = withDefaults(
  defineProps<{
    /** kebab-case lucide icon name, e.g. `settings`. */
    name: string
    size?: IconSize
  }>(),
  {
    size: 'md',
  },
)

const sizeClass = computed(() => sizeMap[props.size])
const iconComponent = computed(() => icons[props.name] ?? null)
</script>

<template>
  <component :is="iconComponent" v-if="iconComponent" :class="sizeClass" aria-hidden="true" />
</template>
