export const MAX_RECENT_PROJECTS = 5;

export function normalizeLocale(locale) {
  return locale === "en" ? "en" : "ru";
}

export function parseStoredList(raw) {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item) => typeof item === "string" && item.trim() !== "");
  } catch {
    return [];
  }
}

export function parseRecentProjects(raw) {
  return parseStoredList(raw);
}

export function rememberRecentProject(existing, nextPath, maxRecentProjects = MAX_RECENT_PROJECTS) {
  const clean = typeof nextPath === "string" ? nextPath.trim() : "";
  if (!clean) return existing.slice(0, maxRecentProjects);
  const deduped = [clean, ...existing.filter((item) => item !== clean)];
  return deduped.slice(0, maxRecentProjects);
}

export function summarizeProjectPath(pathValue) {
  const value = (pathValue || "").trim();
  if (!value) return ".";
  const parts = value.split("/").filter(Boolean);
  if (parts.length <= 2) return value;
  return `.../${parts.slice(-2).join("/")}`;
}

export function parseClientLogs(raw, maxClientLogs = 100) {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter((item) => item && typeof item.message === "string")
      .slice(-maxClientLogs);
  } catch {
    return [];
  }
}

export function escapeHTML(value) {
  if (value == null) return "";
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function excerpt(text, limit = 140) {
  const clean = String(text || "").replace(/\s+/g, " ").trim();
  if (clean.length <= limit) return clean;
  return `${clean.slice(0, Math.max(0, limit - 1))}…`;
}
