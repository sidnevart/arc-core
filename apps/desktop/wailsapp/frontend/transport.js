export function resolveWailsAppBridge(root = globalThis) {
  const candidates = [
    root?.go?.wailsapp?.App,
    root?.go?.main?.App,
  ];
  for (const candidate of candidates) {
    if (candidate && typeof candidate === "object") {
      return candidate;
    }
  }
  return null;
}

export async function waitForWailsBridge(root = globalThis, timeoutMs = 4000, intervalMs = 50) {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    const bridge = resolveWailsAppBridge(root);
    if (bridge) {
      return bridge;
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }
  throw new Error("Wails bridge unavailable");
}

export async function callWailsBridge(root, method, ...args) {
  const bridge = await waitForWailsBridge(root);
  if (!bridge || typeof bridge[method] !== "function") {
    throw new Error("Wails bridge unavailable");
  }
  return bridge[method](...args);
}
