// Types for the plain-JS fixtures shared by the screenshot script and the tests.
export function fixtureFor(
  method: string,
  pathname: string,
  search: string,
): { status?: number; json: unknown } | null
