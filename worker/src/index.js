import { handleImageProxy } from "./image-proxy.js";
import { redirectFor } from "./redirects.js";
import { handleSitemapDiagnostic } from "./site-diagnostics.js";
import { handleStaticSite } from "./static-site.js";
import { handleUpload } from "./upload.js";

export default {
  fetch(request, env) {
    const url = new URL(request.url);

    if (url.pathname === "/api/upload") {
      return handleUpload(request, env);
    }

    if (url.pathname === "/api/image-proxy") {
      return handleImageProxy(request);
    }

    if (url.pathname === "/api/debug/sitemap") {
      return handleSitemapDiagnostic(request, env);
    }

    if (url.pathname === "/api/debug/redirect") {
      const candidate = new URL(url.searchParams.get("path") || "/", url);
      const redirect = redirectFor(candidate.toString());
      return Response.json(
        {
          path: candidate.pathname,
          status: redirect?.status || null,
          location: redirect?.headers.get("location") || null,
        },
        { headers: { "cache-control": "no-store" } },
      );
    }

    if (url.pathname === "/sitemap.xml" && url.searchParams.has("finder-debug")) {
      return handleSitemapDiagnostic(request, env);
    }

    if (env.SITE) {
      return handleStaticSite(request, env);
    }

    if (env.ASSETS) {
      return env.ASSETS.fetch(request);
    }

    return new Response("Not Found", {
      status: 404,
      headers: {
        "content-type": "text/plain; charset=utf-8",
        "cache-control": "no-store",
      },
    });
  },
};
