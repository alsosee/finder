import { handleImageProxy } from "./image-proxy.js";
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
