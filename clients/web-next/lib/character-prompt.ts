import type { CharacterDraft } from "./types";

type CharacterPromptSource = Pick<CharacterDraft, "name" | "description" | "identity_notes">;

/**
 * Deterministic prompt for one character reference image, built from the live
 * form values so nothing is stored twice. The instructions are English like the
 * sheet compiler; the name, description and identity notes are authored content
 * (often Vietnamese) and are copied verbatim, never translated.
 */
export function buildCharacterPrompt(character: CharacterPromptSource): string {
  const name = character.name.trim().replace(/\.+$/, "");
  const description = character.description.trim().replace(/\.+$/, "");
  const identity = character.identity_notes.trim();
  const lines = [
    "Task: draw one character reference image of a single character, as a full-body character study. The face and identifying features must be clearly visible; this is an identity reference, not a sticker sheet.",
  ];
  const subject = [name, description].filter(Boolean).join(". ");
  if (subject) lines.push(`Character: ${subject}.`);
  if (identity) lines.push(`Identity features to keep exactly as described: ${identity}`);
  lines.push(
    "Composition: one character only, standing full body, neutral relaxed pose, facing the viewer, fully visible with even empty margin on every side.",
    "Background: one uniform flat pure white (#FFFFFF); no scenery, ground, floor, sky, shadow, gradient or vignette.",
    "Style: clean flat colours with even outlines; keep the proportions, colours and features described above; do not redesign the character.",
    "Do not draw: text, labels, captions, letters, numbers, grid lines, collage panels, multiple views, watermark or signature.",
  );
  return lines.join("\n");
}
