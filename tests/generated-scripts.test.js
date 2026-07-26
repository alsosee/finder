const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const scriptPath = process.argv[2] || path.join("output", "scripts.js");

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
