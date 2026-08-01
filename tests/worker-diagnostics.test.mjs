import assert from "node:assert/strict";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

async function importWorker() {
  const tempDirectory = await fs.mkdtemp(path.join(os.tmpdir(), "finder-worker-"));
  const sourceDirectory = path.join(repositoryRoot, "worker", "src");
  const workerDirectory = path.join(tempDirectory, "worker", "src");

  await fs.mkdir(workerDirectory, { recursive: true });
  for (const entry of await fs.readdir(sourceDirectory)) {
    if (entry.endsWith(".js")) {
      await fs.copyFile(path.join(sourceDirectory, entry), path.join(workerDirectory, entry));
    }
  }
  await fs.writeFile(
    path.join(workerDirectory, "redirects.generated.js"),
    "export const HAS_REDIRECTS = false;\nexport const REDIRECTS = [];\n",
  );
  await fs.writeFile(path.join(tempDirectory, "package.json"), '{"type":"module"}\n');

  return import(pathToFileURL(path.join(workerDirectory, "index.js")));
}

test("sitemap diagnostic reports missing SITE binding", async () => {
  const worker = await importWorker();
  const response = await worker.default.fetch(new Request("https://example.com/api/debug/sitemap"), {});
  const body = await response.json();

  assert.equal(response.status, 200);
  assert.equal(response.headers.get("content-type"), "application/json; charset=utf-8");
  assert.deepEqual(body, {
    key: "sitemap.xml",
    candidate_keys: ["sitemap.xml"],
    bucket_configured: false,
    found: false,
  });
});

test("sitemap diagnostic checks sitemap.xml in SITE binding", async () => {
  const worker = await importWorker();
  const uploaded = new Date("2026-08-01T12:34:56.000Z");
  const requestedKeys = [];
  const object = {
    size: 123,
    etag: "abc123",
    httpEtag: '"abc123"',
    uploaded,
    httpMetadata: { contentType: "application/xml; charset=utf-8" },
    customMetadata: { sha256: "deadbeef" },
    writeHttpMetadata(headers) {
      headers.set("content-type", this.httpMetadata.contentType);
    },
    body: null,
  };
  const env = {
    SITE: {
      async head(key) {
        requestedKeys.push(["head", key]);
        return object;
      },
      async get(key) {
        requestedKeys.push(["get", key]);
        return object;
      },
    },
  };

  const response = await worker.default.fetch(new Request("https://example.com/api/debug/sitemap"), env);
  const body = await response.json();

  assert.deepEqual(requestedKeys, [
    ["head", "sitemap.xml"],
    ["get", "sitemap.xml"],
    ["get", "sitemap.xml"],
  ]);
  assert.equal(response.status, 200);
  assert.equal(body.key, "sitemap.xml");
  assert.deepEqual(body.candidate_keys, ["sitemap.xml"]);
  assert.equal(body.bucket_configured, true);
  assert.equal(body.found, true);
  assert.equal(body.head_found, true);
  assert.equal(body.get_found, true);
  assert.equal(body.static_status, 200);
  assert.equal(body.static_content_type, "application/xml; charset=utf-8");
  assert.equal(body.size, 123);
  assert.equal(body.uploaded, "2026-08-01T12:34:56.000Z");
  assert.deepEqual(body.http_metadata, { contentType: "application/xml; charset=utf-8" });
  assert.deepEqual(body.custom_metadata, { sha256: "deadbeef" });
});

test("sitemap path can return the diagnostic with finder-debug query", async () => {
  const worker = await importWorker();
  const env = {
    SITE: {
      async head() {
        return null;
      },
      async get() {
        return null;
      },
    },
  };

  const response = await worker.default.fetch(new Request("https://example.com/sitemap.xml?finder-debug=1"), env);
  const body = await response.json();

  assert.equal(response.status, 200);
  assert.equal(body.key, "sitemap.xml");
  assert.equal(body.bucket_configured, true);
  assert.equal(body.head_found, false);
  assert.equal(body.get_found, false);
});
