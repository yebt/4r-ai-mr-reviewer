import type { APIRoute } from "astro";

// A single-page landing needs one URL; generated so <loc> tracks `site`.
export const GET: APIRoute = ({ site }) => {
  const base = (site?.href ?? "https://4r.example.com/").replace(/\/?$/, "/");
  const body =
    `<?xml version="1.0" encoding="UTF-8"?>\n` +
    `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n` +
    `  <url><loc>${base}</loc><changefreq>weekly</changefreq><priority>1.0</priority></url>\n` +
    `</urlset>\n`;
  return new Response(body, {
    headers: { "Content-Type": "application/xml; charset=utf-8" },
  });
};
