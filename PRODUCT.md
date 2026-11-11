# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Existing: Next.js 16 (App Router, `output: "export"` static), React 19, Tailwind CSS v4, TanStack Query. The Go service serves the exported files and the REST API from one HTTPS origin; no Node runtime in production. Keep this stack; no second component library.

## Users

One owner-operator (Vietnamese-speaking) who runs the sticker factory alone. Works on desktop most of the time, switching between Restick Studio and an external AI image tool, but mobile use is equally important (confirmed by the owner: "all of them"): reviewing packs, copying a prompt, uploading a sheet from a phone. A second, indirect audience: client apps (Flutter SDK) that use API keys to read and download packs; they never see this UI.

## Product Purpose

Restick Studio is the single-operator control room of a sticker factory. The loop it serves: compose a reusable-fragment prompt (library + character + palette) → copy it → generate one large image of many stickers in an external AI tool → upload that sheet back → Restick cuts, outlines and stores the stickers → organize them into packs → client apps fetch packs with API keys. Success: the operator moves from idea to a downloadable pack with no lost work and no dead ends.

## Positioning

Not a community marketplace and not an AI image generator. It is a private, opinionated production pipeline: prompt composition + sheet cutting + pack catalog, with API-key distribution to the owner's own apps.

## Operating Context

- Routes: `/` (redirect), `/login`, `/factory` (Studio: compose prompts, upload sheet, process), `/library` (fragment library, groups, palettes), `/characters` (identity profiles and reference images), `/packs`, `/packs/new`, `/packs/detail`, `/keys` (client API keys), `/api-docs` (embeds the Go-served `/docs`).
- Single owner session cookie (`__Host-restick_admin`, HttpOnly); no registration, no contributors, no moderation.
- Image processing is one durable job at a time; stickers are fixed 512×512, thumbnails 128×128.
- UI language is Vietnamese (i18n via `lib/i18n-messages.ts`); keep it.

## Capabilities and Constraints

- Preserve every existing capability, API contract, data shape, and Vietnamese copy meaning. Redesign is presentation and interaction architecture only.
- Static export: no server components needing a runtime, no Next image optimization, no middleware.
- Reference images are admin-only; Restick never sends them to an AI and never fetches external URLs. The UI reminds the operator to attach references by hand.
- API key secrets are shown once at creation; never put them in URLs or storage.
- Restick does not generate images, bill, or edit pixels manually.

## Brand Commitments

Name: Restick. Vietnamese voice. No existing logo asset beyond the wordmark in the shell. No other binding brand commitments.

## Evidence on Hand

Real spec and research: `docs/specs/2026-11-10-single-operator-sticker-factory-design.md`, `docs/research/sticker-factory/`. Seed library of 425 visual fragments, 329 context fragments, 24 palettes. No testimonials, metrics, or customers exist; do not fabricate any.

## Product Principles

1. The pipeline is the product: every screen answers "where am I in idea → prompt → sheet → pack → client".
2. Operator speed over ceremony: frequent actions (copy prompt, upload sheet, add fragment) are one gesture away and keyboard-reachable.
3. Never lose work, never hide state: processing, failure, and recovery are always visible and actionable.
4. Show the stickers: real artwork is the visual centerpiece; the chrome recedes.
5. Destructive and secret-bearing actions are explicit, confirmed, and reversible where the backend allows.

## Accessibility & Inclusion

WCAG 2.2 AA target: visible focus, keyboard operation for all controls, 4.5:1 text contrast, 44px touch targets on mobile, `prefers-reduced-motion` respected. Vietnamese diacritics must render crisply in every font choice (font must support the Vietnamese subset).
