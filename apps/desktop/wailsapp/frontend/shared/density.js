export const CHAT_SCALE_EVENT = "arc:chat-scale";
export const OPEN_DISPLAY_EVENT = "arc:open-display";

const CHAT_SCALE_MIN = 60;
const CHAT_SCALE_MAX = 130;
const CHAT_SCALE_STEP = 5;

const BASE = {
  chatRailWidth: [220, 20, 268],
  topbarPaddingY: 12,
  topbarPaddingX: 14,
  topbarGap: 12,
  compactInputWidth: 136,
  projectChipPaddingY: 6,
  projectChipPaddingX: 10,
  projectSubtitleSize: 0.84,
  agentControlMinWidth: 146,
  controlPaddingY: 7,
  controlPaddingX: 10,
  controlLabelSize: 0.67,
  controlMetaSize: 0.71,
  threadTitleSize: 1.04,
  threadScrollGap: 8,
  messageGap: 8,
  bubblePaddingY: 11,
  bubblePaddingX: 13,
  bubbleRadius: 16,
  bubbleFontSize: 0.92,
  bubbleLineHeight: 1.58,
  bubbleMaxWidth: [920, 88],
  composerPaddingTop: 10,
  composerMarginTop: 8,
  composerTextAreaMinHeight: 104,
  buttonPaddingY: 9,
  buttonPaddingX: 12,
};

function scaled(value, factor) {
  return Math.round(value * factor * 100) / 100;
}

export function normalizeChatScalePercent(value) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return 100;
  const clamped = Math.min(CHAT_SCALE_MAX, Math.max(CHAT_SCALE_MIN, numeric));
  return Math.round(clamped / CHAT_SCALE_STEP) * CHAT_SCALE_STEP;
}

export function applyChatScale(root = globalThis, value = 100) {
  const next = normalizeChatScalePercent(value);
  const factor = next / 100;
  const style = root?.document?.body?.style;
  if (!style) return next;
  style.setProperty("--chat-scale-percent", String(next));
  style.setProperty("--chat-scale-factor", String(factor));
  style.setProperty("--chat-rail-width", `clamp(${scaled(BASE.chatRailWidth[0], factor)}px, ${BASE.chatRailWidth[1]}vw, ${scaled(BASE.chatRailWidth[2], factor)}px)`);
  style.setProperty("--topbar-padding-y", `${scaled(BASE.topbarPaddingY, factor)}px`);
  style.setProperty("--topbar-padding-x", `${scaled(BASE.topbarPaddingX, factor)}px`);
  style.setProperty("--topbar-gap", `${scaled(BASE.topbarGap, factor)}px`);
  style.setProperty("--compact-input-width", `${scaled(BASE.compactInputWidth, factor)}px`);
  style.setProperty("--project-chip-padding-y", `${scaled(BASE.projectChipPaddingY, factor)}px`);
  style.setProperty("--project-chip-padding-x", `${scaled(BASE.projectChipPaddingX, factor)}px`);
  style.setProperty("--project-subtitle-size", `${scaled(BASE.projectSubtitleSize, factor)}rem`);
  style.setProperty("--agent-control-min-width", `${scaled(BASE.agentControlMinWidth, factor)}px`);
  style.setProperty("--control-padding-y", `${scaled(BASE.controlPaddingY, factor)}px`);
  style.setProperty("--control-padding-x", `${scaled(BASE.controlPaddingX, factor)}px`);
  style.setProperty("--control-label-size", `${scaled(BASE.controlLabelSize, factor)}rem`);
  style.setProperty("--control-meta-size", `${scaled(BASE.controlMetaSize, factor)}rem`);
  style.setProperty("--thread-title-size", `${scaled(BASE.threadTitleSize, factor)}rem`);
  style.setProperty("--thread-scroll-gap", `${scaled(BASE.threadScrollGap, factor)}px`);
  style.setProperty("--message-gap", `${scaled(BASE.messageGap, factor)}px`);
  style.setProperty("--bubble-padding-y", `${scaled(BASE.bubblePaddingY, factor)}px`);
  style.setProperty("--bubble-padding-x", `${scaled(BASE.bubblePaddingX, factor)}px`);
  style.setProperty("--bubble-radius", `${scaled(BASE.bubbleRadius, factor)}px`);
  style.setProperty("--bubble-font-size", `${scaled(BASE.bubbleFontSize, factor)}rem`);
  style.setProperty("--bubble-line-height", `${scaled(BASE.bubbleLineHeight, 1)}`);
  style.setProperty("--bubble-max-width", `min(${scaled(BASE.bubbleMaxWidth[0], factor)}px, ${BASE.bubbleMaxWidth[1]}%)`);
  style.setProperty("--composer-padding-top", `${scaled(BASE.composerPaddingTop, factor)}px`);
  style.setProperty("--composer-margin-top", `${scaled(BASE.composerMarginTop, factor)}px`);
  style.setProperty("--composer-textarea-min-height", `${scaled(BASE.composerTextAreaMinHeight, factor)}px`);
  style.setProperty("--button-padding-y", `${scaled(BASE.buttonPaddingY, factor)}px`);
  style.setProperty("--button-padding-x", `${scaled(BASE.buttonPaddingX, factor)}px`);
  return next;
}

export function bindNativeChatScaleEvents(root = globalThis, onScale = null, onOpenDisplay = null) {
  const runtime = root?.runtime;
  if (!runtime || typeof runtime.EventsOn !== "function") {
    return () => {};
  }
  const unsubscribeScale = runtime.EventsOn(CHAT_SCALE_EVENT, (...data) => {
    const next = applyChatScale(root, data[0]);
    if (typeof onScale === "function") onScale(next);
  });
  const unsubscribeDisplay = runtime.EventsOn(OPEN_DISPLAY_EVENT, () => {
    if (typeof onOpenDisplay === "function") onOpenDisplay();
  });
  return () => {
    if (typeof unsubscribeScale === "function") unsubscribeScale();
    if (typeof unsubscribeDisplay === "function") unsubscribeDisplay();
  };
}

export const CHAT_SCALE_LIMITS = {
  min: CHAT_SCALE_MIN,
  max: CHAT_SCALE_MAX,
  step: CHAT_SCALE_STEP,
};
