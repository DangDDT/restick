// External image tools the operator pastes the compiled prompt into. URLs verified 2026-11-11;
// no free-tier/pricing data on purpose (goes stale). Order: sticker-sheet fit first.
export type AiProvider = { id: string; name: string; url: string; hint: string };

export const AI_PROVIDERS: readonly AiProvider[] = [
  { id: "chatgpt", name: "ChatGPT", url: "https://chatgpt.com", hint: "GPT Image, bám prompt tốt" },
  { id: "gemini", name: "Gemini", url: "https://gemini.google.com/app", hint: "Nano Banana" },
  { id: "ideogram", name: "Ideogram", url: "https://ideogram.ai", hint: "Chữ rõ, có nền trong suốt" },
  { id: "leonardo", name: "Leonardo", url: "https://app.leonardo.ai", hint: "Có nền trong suốt" },
  { id: "midjourney", name: "Midjourney", url: "https://www.midjourney.com", hint: "Nét minh họa nghệ thuật" },
  { id: "firefly", name: "Adobe Firefly", url: "https://firefly.adobe.com", hint: "Adobe" },
  { id: "recraft", name: "Recraft", url: "https://www.recraft.ai", hint: "Minh họa, vector" },
  { id: "flux", name: "Flux", url: "https://playground.bfl.ai", hint: "Black Forest Labs" },
  { id: "kling", name: "Kling", url: "https://kling.ai/app", hint: "Thiên về video, có tạo ảnh" },
  { id: "runway", name: "Runway", url: "https://app.runwayml.com", hint: "Thiên về video, có tạo ảnh" },
];
