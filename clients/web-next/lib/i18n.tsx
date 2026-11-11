import { vi, type MessageKey } from "./i18n-messages";

export type { MessageKey };

// Single Vietnamese catalog; the studio has one interface language and the
// English text lives in the prompt content the operator authors.
export function t(key: MessageKey): string {
  return vi[key];
}
