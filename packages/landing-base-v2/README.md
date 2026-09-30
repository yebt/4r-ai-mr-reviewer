# landing-base-v2

The 4R marketing landing ("the path of one MR"). Plain Astro + CSS, sharing
spa-b's identity: `src/styles/tokens.css` is a port of
`packages/spa-b/src/shared/assets/tokens.css` (light + dark), and the theme uses
the same `4r-color-scheme` storage key and `html.light` / `html.dark` classes.

```sh
bun install
bun run dev     # or, from the repo root: make run-landing-v2
bun run build   # or: make build-landing-v2
```

Build-time env:

| Variable | Purpose |
|---|---|
| `SITE_URL` | Production origin for canonical, sitemap, and Open Graph URLs |
| `PUBLIC_DOCS_URL` | Docs link target (defaults to `/docs`) |

Screenshots live in `public/shots/` as `<name>-light.webp` / `<name>-dark.webp`
(`review`, `publish`, `flow`, `run`), captured from spa-b with synthetic data.
