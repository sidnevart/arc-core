import test from "node:test";
import assert from "node:assert/strict";

import { messagesFor, translate } from "../wailsapp/frontend/shared/messages.js";
import { applyChatScale, normalizeChatScalePercent } from "../wailsapp/frontend/shared/density.js";
import { renderMarkdown } from "../wailsapp/frontend/shared/markdown.js";
import { parseClientLogs, parseStoredList } from "../wailsapp/frontend/shared/ux.js";

test("messagesFor applies preview-specific overrides without losing base keys", () => {
  const nativeMessages = messagesFor("native");
  const previewMessages = messagesFor("preview");

  assert.equal(previewMessages.ru["settings.logs"], "Логи preview");
  assert.equal(previewMessages.en["settings.logs"], "Preview logs");
  assert.equal(nativeMessages.ru["settings.logs"], "Логи интерфейса");
  assert.equal(translate(previewMessages, "ru", "chat.send"), "Отправить");
  assert.equal(translate(previewMessages, "en", "chat.send"), "Send");
});

test("shared parsers stay deterministic", () => {
  assert.deepEqual(parseStoredList(JSON.stringify(["/a", "", 1, "/b"])), ["/a", "/b"]);

  const logs = parseClientLogs(JSON.stringify([
    { level: "info", message: "one" },
    { level: "warn", message: "two" },
    { level: "broken" },
  ]), 10);
  assert.deepEqual(logs, [
    { level: "info", message: "one" },
    { level: "warn", message: "two" },
  ]);
});

test("chat scale helpers normalize and apply DOM state", () => {
  assert.equal(normalizeChatScalePercent(59), 60);
  assert.equal(normalizeChatScalePercent(108), 110);
  assert.equal(normalizeChatScalePercent(131), 130);
  const root = {
    document: {
      body: {
        style: {
          values: {},
          setProperty(name, value) {
            this.values[name] = value;
          },
        },
      },
    },
  };
  assert.equal(applyChatScale(root, 110), 110);
  assert.equal(root.document.body.style.values["--chat-scale-percent"], "110");
  assert.match(root.document.body.style.values["--chat-rail-width"], /clamp/);
});

test("markdown renderer formats headings, lists, quotes, and code fences", () => {
  const html = renderMarkdown(`# Title

- One
- Two

> Quote

\`\`\`js
console.log("ok")
\`\`\``);

  assert.match(html, /<h3>Title<\/h3>/);
  assert.match(html, /<ul><li>One<\/li><li>Two<\/li><\/ul>/);
  assert.match(html, /<blockquote><p>Quote<\/p><\/blockquote>/);
  assert.match(html, /<pre class="code-block"><code>console\.log\(&quot;ok&quot;\)<\/code><\/pre>/);
});
