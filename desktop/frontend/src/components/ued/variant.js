/**
 * Shared variant/size/tone normalization for UED components.
 *
 * Historically every component (UButton, UTag, UAlert, UIconButton, message)
 * re-implemented its own `danger -> error` mapping and size whitelist.
 * These helpers centralize that logic so semantics stay consistent.
 */

// `danger` is accepted everywhere as a friendly alias of `error`.
const DANGER_ALIASES = new Set(["danger"]);

/**
 * Normalize a variant/tone string.
 *
 * - Lowercases the input.
 * - Maps the `danger` alias to `error`.
 * - When `allowed` is provided, falls back to `fallback` for any value
 *   outside the set (keeps unknown props from producing unstyled classes).
 *
 * @param {string} value      Raw prop value.
 * @param {string} fallback   Value used when input is empty or invalid.
 * @param {string[]=} allowed Optional whitelist of valid outcomes (post-alias).
 * @returns {string}
 */
export function normalizeVariant(value, fallback, allowed) {
  let next = String(value || fallback).toLowerCase();
  if (DANGER_ALIASES.has(next)) {
    next = "error";
  }
  if (allowed && !allowed.includes(next)) {
    next = fallback;
  }
  return next;
}

/**
 * Normalize a size string against a whitelist, falling back when invalid.
 *
 * @param {string} value    Raw prop value.
 * @param {string} fallback Default size.
 * @param {string[]} allowed Valid sizes.
 * @returns {string}
 */
export function normalizeSize(value, fallback, allowed) {
  const next = String(value || fallback).toLowerCase();
  return allowed.includes(next) ? next : fallback;
}

// Whitelists reused across components.
export const BUTTON_VARIANTS = ["primary", "secondary", "success", "warning", "error", "info", "ghost"];
export const TAG_VARIANTS = ["primary", "success", "warning", "error", "info", "neutral"];
export const ALERT_TYPES = ["success", "info", "warning", "error"];
export const MESSAGE_TYPES = ["success", "info", "warning", "error", "loading"];
export const ICON_BUTTON_VARIANTS = ["secondary", "primary", "info", "success", "warning", "error", "ghost"];

export const CONTROL_SIZES = ["xs", "sm", "md", "lg"];
export const LOADING_SIZES = ["sm", "md", "lg"];
