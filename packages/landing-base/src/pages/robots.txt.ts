import type { APIRoute } from "astro";

// Generated at build time so the Sitemap line tracks `site` (astro.config).
export const GET: APIRoute = ({ site }) => {
  const sitemap = site ? new URL("sitemap.xml", site).href : "/sitemap.xml";
  return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${sitemap}\n`, {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
};
