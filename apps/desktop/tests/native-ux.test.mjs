import test from "node:test";
import assert from "node:assert/strict";

import {
  MAX_RECENT_PROJECTS,
  normalizeLocale,
  parseRecentProjects,
  rememberRecentProject,
  summarizeProjectPath,
} from "../wailsapp/frontend/ux.js";

test("normalizeLocale defaults to ru", () => {
  assert.equal(normalizeLocale("fr"), "ru");
  assert.equal(normalizeLocale("ru"), "ru");
  assert.equal(normalizeLocale("en"), "en");
});

test("parseRecentProjects tolerates bad data", () => {
  assert.deepEqual(parseRecentProjects(""), []);
  assert.deepEqual(parseRecentProjects("{bad json"), []);
  assert.deepEqual(parseRecentProjects(JSON.stringify(["/a", "", 1, "/b"])), ["/a", "/b"]);
});

test("rememberRecentProject deduplicates and caps the list", () => {
  let paths = ["/a", "/b", "/c", "/d", "/e"];
  paths = rememberRecentProject(paths, "/c");
  assert.deepEqual(paths, ["/c", "/a", "/b", "/d", "/e"]);
  paths = rememberRecentProject(paths, "/f");
  assert.equal(paths.length, MAX_RECENT_PROJECTS);
  assert.deepEqual(paths, ["/f", "/c", "/a", "/b", "/d"]);
});

test("summarizeProjectPath shortens long paths", () => {
  assert.equal(summarizeProjectPath("/Users/artem/project"), ".../artem/project");
  assert.equal(summarizeProjectPath("/tmp/app"), "/tmp/app");
  assert.equal(summarizeProjectPath(""), ".");
});
