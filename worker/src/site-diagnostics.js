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
      bucket_configured: false,
      found: false,
    });
  }

  const object = await bucket.head("sitemap.xml");
  if (!object) {
    return jsonResponse({
      key: "sitemap.xml",
      bucket_configured: true,
      found: false,
    });
  }

  return jsonResponse({
    key: "sitemap.xml",
    bucket_configured: true,
    found: true,
    size: object.size,
    etag: object.etag,
    http_etag: object.httpEtag,
    uploaded: object.uploaded?.toISOString(),
    http_metadata: object.httpMetadata,
    custom_metadata: object.customMetadata,
  });
}
