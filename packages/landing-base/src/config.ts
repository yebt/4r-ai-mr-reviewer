// Outbound links used across the landing. Point `docs` at the deployed docs
// site (packages/documentation) via PUBLIC_DOCS_URL at build time; it defaults
// to a monorepo-friendly /docs path.
export const SITE = {
  name: "4R",
  tagline: "AI Merge Request Reviewer",
  github: "https://github.com/yebt/4r-ai-mr-reviewer",
  docs: import.meta.env.PUBLIC_DOCS_URL ?? "/docs",
};
