import test from "node:test";
import assert from "node:assert/strict";

import { callWailsBridge, resolveWailsAppBridge, waitForWailsBridge } from "../wailsapp/frontend/transport.js";

test("resolveWailsAppBridge prefers wailsapp namespace", () => {
  const bridge = { Home: () => "ok" };
  const root = {
    go: {
      wailsapp: { App: bridge },
      main: { App: { Home: () => "wrong" } },
    },
  };
  assert.equal(resolveWailsAppBridge(root), bridge);
});

test("resolveWailsAppBridge falls back to main namespace", () => {
  const bridge = { Home: () => "ok" };
  const root = { go: { main: { App: bridge } } };
  assert.equal(resolveWailsAppBridge(root), bridge);
});

test("callWailsBridge invokes method from resolved bridge", async () => {
  const root = {
    go: {
      wailsapp: {
        App: {
          Home: async (path) => ({ path }),
        },
      },
    },
  };
  const result = await callWailsBridge(root, "Home", "/tmp/project");
  assert.deepEqual(result, { path: "/tmp/project" });
});

test("callWailsBridge throws when no bridge is available", async () => {
  await assert.rejects(() => callWailsBridge({}, "Home", "."), /Wails bridge unavailable/);
});

test("waitForWailsBridge resolves when bridge appears later", async () => {
  const root = {};
  setTimeout(() => {
    root.go = {
      wailsapp: {
        App: {
          Home: async () => ({ ok: true }),
        },
      },
    };
  }, 25);
  const bridge = await waitForWailsBridge(root, 500, 10);
  assert.equal(typeof bridge.Home, "function");
});
