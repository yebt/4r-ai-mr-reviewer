// Shared types for the reusable FilterBuilder (a Linear-style progressive filter
// bar). Kept in a plain .ts module so both the component and its consumers
// (findings triage, reviews list) import the same shapes without a .vue type
// dependency.

/** One selectable value within a filter field. */
export interface FilterOption {
  value: string
  label: string
}

/**
 * A filterable field the builder can add as a chip. `multi` renders a checkbox
 * list (any-of semantics); otherwise a single-select list that closes on pick.
 */
export interface FilterField {
  key: string
  label: string
  multi?: boolean
  options: FilterOption[]
}

/** An active filter chip: a field key plus its currently selected values. */
export interface ActiveFilter {
  key: string
  values: string[]
}
