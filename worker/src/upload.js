function jsonResponse(body, status = 200, headers = {}) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      "cache-control": "no-store",
      ...headers,
    },
  });
}

export async function handleUpload(request, env) {
  if (request.method !== "PUT") {
    return jsonResponse(
      { error: { message: "Method Not Allowed" } },
      405,
      { allow: "PUT", xerror: `Method ${request.method} is not allowed` },
    );
  }

  try {
    const rawKey = request.headers.get("x-file-name");
    if (!rawKey) {
      return jsonResponse({ error: "Missing x-file-name header" }, 400);
    }

    const key = decodeURIComponent(rawKey);
    if (!key) {
      return jsonResponse({ error: "Missing x-file-name header" }, 400);
    }

    if (env.MEDIA) {
      await env.MEDIA.put(key, request.body);
    } else {
      const uploadURL = env.LOCAL_UPLOAD_URL || "http://localhost:8780/upload";
      const uploaderResponse = await fetch(uploadURL, {
        method: "POST",
        body: request.body,
        headers: {
          "x-file-name": key,
        },
      });

      if (uploaderResponse.status !== 201) {
        const text = await uploaderResponse.text();
        throw new Error(
          `Local server failed with status code ${uploaderResponse.status}: ${text}`,
        );
      }
    }

    if (!env.GHP_TOKEN || !env.GITHUB_REPO) {
      return jsonResponse({
        status: "ok",
        key,
        actions_triggered: false,
      });
    }

    const dispatchResponse = await fetch(
      `https://api.github.com/repos/${env.GITHUB_REPO}/dispatches`,
      {
        method: "POST",
        headers: {
          authorization: `Bearer ${env.GHP_TOKEN}`,
          accept: "application/vnd.github.everest-preview+json",
          "content-type": "application/json",
          "user-agent": "alsosee/finder/1.0.0 (Cloudflare Worker)",
        },
        body: JSON.stringify({
          event_type: "pull",
          client_payload: {
            path: key,
            trigger: "upload",
          },
        }),
      },
    );

    if (dispatchResponse.status !== 204) {
      const text = await dispatchResponse.text();
      console.log(dispatchResponse.headers);
      console.log(text);
      throw new Error(
        `GitHub API failed with status code ${dispatchResponse.status}`,
      );
    }

    return jsonResponse({ status: "ok", key });
  } catch (err) {
    return jsonResponse({ error: err.stack || String(err) }, 500);
  }
}
