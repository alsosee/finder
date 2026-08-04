const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const scriptPath = process.env.SCRIPT_PATH || process.argv[2] || path.join("output", "scripts.js");

function generatedScript() {
  try {
    return fs.readFileSync(scriptPath, "utf8");
  } catch (error) {
    if (error.code === "ENOENT") {
      throw new Error(`Generated script not found at ${scriptPath}. Run generation before this test.`);
    }
    throw error;
  }
}

function scriptFunction(script, name) {
  const start = script.indexOf(`function ${name}(`);
  assert.notEqual(start, -1, `function ${name} not found`);

  const next = script.indexOf("\n    function ", start + 1);
  if (next === -1) {
    return script.slice(start);
  }

  return script.slice(start, next);
}

test("restoreFocus only restores global panel scroll in columns view", () => {
  const restoreFocus = scriptFunction(generatedScript(), "restoreFocus");

  const columnGuard = restoreFocus.indexOf('if (view === "columns")');
  assert.notEqual(columnGuard, -1, "restoreFocus should guard global panel scroll restoration to columns view");

  const panelScroll = restoreFocus.indexOf("panel.scrollTop = scrollPostions[index];");
  assert.notEqual(panelScroll, -1, "restoreFocus should still restore panel offsets for columns view");
  assert.ok(panelScroll > columnGuard, "restoreFocus restores global panel scroll before checking columns view");

  assert.ok(
    restoreFocus.includes("focusWithoutScrolling(link);"),
    "restoreFocus should focus restored links without changing scroll",
  );
});

test("scroll positions are saved from scroll events", () => {
  const script = generatedScript();

  assert.ok(script.includes("function scheduleScrollPositionSave(event)"), "missing debounced scroll save handler");
  assert.ok(
    script.includes("document.addEventListener('scroll', scheduleScrollPositionSave, true);"),
    "scroll save handler is not registered in capture phase",
  );
  assert.ok(script.includes("requestAnimationFrame(function()"), "scroll saves should be throttled");
});

test("search saves folder scroll before replacing panels", () => {
  const search = scriptFunction(generatedScript(), "search");

  const firstSearch = search.indexOf("if (previousQuery.length == 0) {");
  assert.notEqual(firstSearch, -1, "search first-query branch not found");

  const body = search.slice(firstSearch);
  const save = body.indexOf("saveScrollPosition();");
  const backup = body.indexOf("panelsBackup = panels.innerHTML;");

  assert.notEqual(save, -1, "search should save folder scroll before entering search mode");
  assert.notEqual(backup, -1, "search should back up folder panels before entering search mode");
  assert.ok(save < backup, "search backs up/replaces folder panels before saving current scroll");
});

test("displaySearchResults updates DOM before callback", () => {
  const displaySearchResults = scriptFunction(generatedScript(), "displaySearchResults");

  const updateDOM = displaySearchResults.indexOf("panels.innerHTML =");
  const callback = displaySearchResults.indexOf("if (callback) {");

  assert.notEqual(updateDOM, -1, "displaySearchResults does not update panels");
  assert.notEqual(callback, -1, "displaySearchResults does not invoke callback");
  assert.ok(updateDOM < callback, "displaySearchResults invokes callback before search results exist in the DOM");
});

test("shared panels are cached and reprocessed by htmx", () => {
  const script = generatedScript();
  const response = scriptFunction(script, "sharedPanelResponse");
  const hydrate = scriptFunction(script, "hydrateSharedPanel");

  assert.ok(response.includes('fetch(src, {cache: "force-cache"})'), "shared panel fetch should use the browser cache");
  assert.ok(hydrate.includes("htmx.process(panel);"), "links inserted into a shared panel should be processed by htmx");
  assert.ok(
    hydrate.includes('restoreFocus("sharedPanel");'),
    "shared panel hydration should restore keyboard focus and column scroll",
  );
  assert.ok(
    hydrate.includes('updateSharedPanelBreadcrumb(panel.dataset.panelSrc, "loading");'),
    "shared panel hydration should mark the breadcrumb as loading",
  );
  assert.ok(
    hydrate.includes("finishSharedPanelBreadcrumb(panel.dataset.panelSrc);"),
    "shared panel hydration should defer clearing the breadcrumb loading state",
  );
  assert.ok(
    hydrate.includes('updateSharedPanelBreadcrumb(panel.dataset.panelSrc, "error");'),
    "shared panel hydration should expose load failures in the breadcrumb",
  );
});

test("shared panel breadcrumb loaded state waits for a paint", () => {
  const finish = scriptFunction(generatedScript(), "finishSharedPanelBreadcrumb");

  assert.ok(finish.includes("requestAnimationFrame(function()"), "loaded state should wait for a frame");
  assert.ok(finish.includes("hasLoadedSharedPanel(src)"), "loaded state should verify the current panel is loaded");
  assert.ok(
    finish.includes('updateSharedPanelBreadcrumb(src, "loaded");'),
    "loaded state should eventually clear the breadcrumb loading marker",
  );
});

test("shared panel highlighting follows the current pathname", () => {
  const update = scriptFunction(generatedScript(), "updateSharedPanelState");

  assert.ok(update.includes('link.classList.remove("active", "in-path");'), "stale route classes should be removed");
  assert.ok(update.includes("normalizedPathname(link.href) === currentPath"), "matching should use normalized link paths");
  assert.ok(update.includes('currentLink.classList.add("active", "in-path");'), "the current person should be highlighted");
});
