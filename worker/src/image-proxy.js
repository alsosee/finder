const MAX_IMAGE_BYTES = 10 * 1024 * 1024;
const FETCH_TIMEOUT_MS = 10000;
const MAX_REDIRECTS = 5;

const ALLOWED_IMAGE_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/webp",
  "image/gif",
]);

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
    },
  });
}

function parseIPv4(hostname) {
  const parts = hostname.split(".");
  if (parts.length !== 4) {
    return null;
  }

  const bytes = parts.map((part) => {
    if (!/^\d+$/.test(part)) {
      return null;
    }

    const value = Number(part);
    if (value < 0 || value > 255) {
      return null;
    }

    return value;
  });

  if (bytes.some((part) => part === null)) {
    return null;
  }

  return bytes;
}

function isPrivateIPv4(bytes) {
  const [a, b] = bytes;
  return (
    a === 0 ||
    a === 10 ||
    a === 127 ||
    (a === 100 && b >= 64 && b <= 127) ||
    (a === 169 && b === 254) ||
    (a === 172 && b >= 16 && b <= 31) ||
    (a === 192 && b === 0) ||
    (a === 192 && b === 168) ||
    (a === 198 && (b === 18 || b === 19)) ||
    a >= 224
  );
}

function isBlockedHostname(hostname) {
  const host = hostname.toLowerCase().replace(/^\[|\]$/g, "");

  if (
    host === "localhost" ||
    host.endsWith(".localhost") ||
    host === "metadata.google.internal"
  ) {
    return true;
  }

  const ipv4 = parseIPv4(host);
  if (ipv4 && isPrivateIPv4(ipv4)) {
    return true;
  }

  if (
    host === "::1" ||
    host.startsWith("fc") ||
    host.startsWith("fd") ||
    host.startsWith("fe80:")
  ) {
    return true;
  }

  return false;
}

function validateImageURL(rawURL) {
  let url;
  try {
    url = new URL(rawURL);
  } catch {
    throw new Error("Invalid URL");
  }

  if (url.protocol !== "https:") {
    throw new Error("Only HTTPS image URLs are supported");
  }

  if (url.username || url.password) {
    throw new Error("Image URLs must not include credentials");
  }

  if (url.port && url.port !== "443") {
    throw new Error("Image URLs must use the default HTTPS port");
  }

  if (isBlockedHostname(url.hostname)) {
    throw new Error("This image host is not allowed");
  }

  return url;
}

function firstHeaderValue(headers, name) {
  const value = headers.get(name);
  if (!value) {
    return "";
  }

  return value.split(";")[0].trim().toLowerCase();
}

async function readLimitedBody(response) {
  const contentLength = response.headers.get("content-length");
  if (contentLength && Number(contentLength) > MAX_IMAGE_BYTES) {
    throw new Error("Image is too large");
  }

  if (!response.body) {
    throw new Error("Image response body is empty");
  }

  const reader = response.body.getReader();
  const chunks = [];
  let total = 0;

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }

    total += value.byteLength;
    if (total > MAX_IMAGE_BYTES) {
      await reader.cancel();
      throw new Error("Image is too large");
    }

    chunks.push(value);
  }

  return new Blob(chunks, {
    type: firstHeaderValue(response.headers, "content-type"),
  });
}

async function fetchImage(rawURL) {
  let url = validateImageURL(rawURL);

  for (let redirects = 0; redirects <= MAX_REDIRECTS; redirects += 1) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);

    let response;
    try {
      response = await fetch(url.toString(), {
        redirect: "manual",
        signal: controller.signal,
        headers: {
          accept: "image/avif,image/webp,image/png,image/jpeg,image/gif;q=0.9,*/*;q=0.1",
          "user-agent": "alsosee/finder image proxy",
        },
      });
    } finally {
      clearTimeout(timeout);
    }

    if ([301, 302, 303, 307, 308].includes(response.status)) {
      const location = response.headers.get("location");
      if (!location) {
        throw new Error("Image redirect is missing a Location header");
      }

      url = validateImageURL(new URL(location, url).toString());
      continue;
    }

    if (!response.ok) {
      throw new Error(`Image request failed with status ${response.status}`);
    }

    const contentType = firstHeaderValue(response.headers, "content-type");
    if (!ALLOWED_IMAGE_TYPES.has(contentType)) {
      throw new Error("URL did not return a supported image type");
    }

    const blob = await readLimitedBody(response);
    return { blob, contentType, sourceURL: url.toString() };
  }

  throw new Error("Image URL redirected too many times");
}

export async function handleImageProxy(request) {
  if (request.method !== "POST") {
    return jsonResponse({ error: "Method Not Allowed" }, 405);
  }

  try {
    const body = await request.json();
    if (!body || typeof body.url !== "string") {
      return jsonResponse({ error: "Missing image URL" }, 400);
    }

    const image = await fetchImage(body.url);
    return new Response(image.blob, {
      status: 200,
      headers: {
        "content-type": image.contentType,
        "cache-control": "no-store",
        "x-source-url": image.sourceURL,
      },
    });
  } catch (error) {
    const message = error && error.name === "AbortError"
      ? "Image request timed out"
      : error.message || "Failed to fetch image";

    return jsonResponse({ error: message }, 400);
  }
}
