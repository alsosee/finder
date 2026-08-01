import { candidateKeys, handleStaticSite } from "./static-site.js";

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body, null, 2) + "\n", {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
    },
  });
}

export async function handleSitemapDiagnostic(request, env) {
  if (request.method !== "GET" && request.method !== "HEAD") {
    return jsonResponse(
      { error: { message: "Method Not Allowed" } },
      405,
    );
  }

  const bucket = env.SITE;
  if (!bucket) {
    return jsonResponse({
      key: "sitemap.xml",
      candidate_keys: candidateKeys("/sitemap.xml"),
      bucket_configured: false,
      found: false,
    });
  }

  const headObject = await bucket.head("sitemap.xml");
  const getObject = await bucket.get("sitemap.xml");
  const staticResponse = await handleStaticSite(
    new Request(new URL("/sitemap.xml", request.url), { method: "GET" }),
    env,
  );
  await staticResponse.body?.cancel();

  if (!headObject && !getObject) {
    return jsonResponse({
      key: "sitemap.xml",
      candidate_keys: candidateKeys("/sitemap.xml"),
      bucket_configured: true,
      found: false,
      head_found: false,
      get_found: false,
      static_status: staticResponse.status,
      static_content_type: staticResponse.headers.get("content-type"),
      static_etag: staticResponse.headers.get("etag"),
    });
  }

  const object = getObject || headObject;
  return jsonResponse({
    key: "sitemap.xml",
    candidate_keys: candidateKeys("/sitemap.xml"),
    bucket_configured: true,
    found: true,
    head_found: Boolean(headObject),
    get_found: Boolean(getObject),
    static_status: staticResponse.status,
    static_content_type: staticResponse.headers.get("content-type"),
    static_etag: staticResponse.headers.get("etag"),
    size: object.size,
    etag: object.etag,
    http_etag: object.httpEtag,
    uploaded: object.uploaded?.toISOString(),
    http_metadata: object.httpMetadata,
    custom_metadata: object.customMetadata,
  });
}
