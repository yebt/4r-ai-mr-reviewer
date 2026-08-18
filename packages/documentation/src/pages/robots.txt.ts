import type { APIRoute } from "astro";

// Points crawlers at the sitemap Starlight generates (sitemap-index.xml) once
// `site` is set in astro.config.
export const GET: APIRoute = ({ site }) => {
  const sitemap = site ? new URL("sitemap-index.xml", site).href : "/sitemap-index.xml";
  return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${sitemap}\n`, {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
};
