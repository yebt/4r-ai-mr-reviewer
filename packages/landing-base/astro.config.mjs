// @ts-check
import { defineConfig } from 'astro/config';

// `site` is the production origin — it powers canonical URLs, the sitemap, and
// absolute Open Graph image URLs. Set SITE_URL at build time (e.g.
// `SITE_URL=https://4r.yourdomain.com astro build`); the placeholder is only a
// fallback so local builds succeed.
// https://astro.build/config
export default defineConfig({
  site: process.env.SITE_URL ?? 'https://4r.example.com',
});
