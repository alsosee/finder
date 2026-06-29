const TEXT_TYPES = {
  css: "text/css; charset=utf-8",
  html: "text/html; charset=utf-8",
  js: "application/javascript; charset=utf-8",
  json: "application/json; charset=utf-8",
  txt: "text/plain; charset=utf-8",
  webmanifest: "application/manifest+json; charset=utf-8",
};

const BINARY_TYPES = {
  avif: "image/avif",
  ico: "image/x-icon",
  jpeg: "image/jpeg",
  jpg: "image/jpeg",
  png: "image/png",
  svg: "image/svg+xml",
  webp: "image/webp",
};

export async function handleStaticSite(request, env) {
  if (request.method !== "GET" && request.method !== "HEAD") {
    return new Response("Method Not Allowed", {
      status: 405,
      headers: {
        allow: "GET, HEAD",
        "content-type": "text/plain; charset=utf-8",
      },
    });
  }

  const bucket = env.SITE;
  if (!bucket) {
    return new Response("Static site bucket is not configured", {
      status: 500,
      headers: {
        "content-type": "text/plain; charset=utf-8",
        "cache-control": "no-store",
      },
    });
  }

  const match = await findObject(bucket, new URL(request.url).pathname);
  if (match) {
    return objectResponse(match.object, match.key, request);
  }

  const notFound = await bucket.get("404.html");
  if (notFound) {
    return objectResponse(notFound, "404.html", request, 404);
  }

  return new Response("Not Found", {
    status: 404,
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "cache-control": "no-store",
    },
  });
}

async function findObject(bucket, pathname) {
  for (const key of candidateKeys(pathname)) {
    const object = await bucket.get(key);
    if (object) {
      return { key, object };
    }
  }
  return null;
}

export function candidateKeys(pathname) {
  const key = requestPathToKey(pathname);
  if (!key) {
    return ["index.html"];
  }

  const candidates = [key];
  if (key.endsWith("/")) {
    candidates.push(key + "index.html");
  } else if (!hasFileExtension(key)) {
    candidates.push(key + ".html", key + "/index.html");
  }

  return candidates;
}

function requestPathToKey(pathname) {
  const path = pathname.replace(/^\/+/, "");
  if (path === "") {
    return "";
  }

  try {
    return decodeURIComponent(path);
  } catch {
    return path;
  }
}

function hasFileExtension(path) {
  const name = path.split("/").pop() || "";
  return name.includes(".");
}

function objectResponse(object, key, request, status = 200) {
  const headers = new Headers();
  object.writeHttpMetadata(headers);
  headers.set("etag", object.httpEtag);

  if (!headers.has("content-type")) {
    headers.set("content-type", contentType(key));
  }

  return new Response(request.method === "HEAD" ? null : object.body, {
    status,
    headers,
  });
}

function contentType(key) {
  const extension = key.split(".").pop()?.toLowerCase() || "";
  return TEXT_TYPES[extension] || BINARY_TYPES[extension] || "application/octet-stream";
}
