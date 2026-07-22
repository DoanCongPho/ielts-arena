# IELTS Arena — Visual Design System v2.0

A reference for making every screen look like it came from one deliberate design, not a stack of separately-vibe-coded pages. Treat this file as the single source of truth: any AI-generated frontend code should be checked against it, not the other way around.

## 0. Changelog

**v2.0 — the "vibrant SaaS" refresh.** v1.0's flat, hairline-only, "one accent max" system read as *plain* rather than *clean* once it was actually built and looked at side-by-side with the app it was replacing — cards had no depth, and the Dashboard repeated each skill's name twice (a "READING" tag sitting directly next to a button that already said "Luyện tập Reading"). v2.0 keeps everything from v1.0 (the neutrals, the type system, the Band Meter, the spacing scale, the "no third radius value" discipline) and adds:
- **Real elevation.** Cards get a soft ambient shadow by default, not just a hairline border; interactive cards lift further on hover. Shadow is no longer modal-only.
- **Gradient skill icons (§4b, `IconChip`).** Skill identity now has an icon, not just a text tag — this is what replaced the redundant Dashboard tags. Icons carry color via a two-tone gradient (`-500` → `-600`), which is the one place skill color is allowed to feel rich rather than restrained.
- **A gradient primary button.** The one primary action per screen now reads as more considered than a flat fill.

v1.0's core discipline still holds: one deliberate system, not a pile of one-off choices. "Vibrant" means *more considered color and depth*, not *more colors at once* — §1.5's two-accent rule still applies.

## 1. Color

### 1.1 Neutrals (chrome, text, surfaces)

| Token | Hex | Use |
|---|---|---|
| `ink-900` | `#16233F` | Primary text, dark nav/sidebar surface |
| `ink-700` | `#3A4457` | Secondary text |
| `ink-400` | `#7A8296` | Muted text, placeholders, disabled |
| `paper-50` | `#EEF1EE` | Page background |
| `surface-0` | `#FFFFFF` | Cards, modals, inputs |
| `hairline` | `#16233F` at 10% opacity | Borders, dividers |

Deliberate choice: the background is a cool, quiet grey-white, not the warm cream that most AI-generated UIs default to. It reads closer to exam paper than to a lifestyle blog.

### 1.2 Skill identity colors

Each skill module gets one accent (a `-600`, used for text/tags as in v1.0) plus a lighter `-500` companion added in v2.0 for gradient chips. Never use a skill's color for chrome shared across skills (buttons, global nav background, etc.) — that's what keeps four modules feeling like one app.

| Skill | `-500` (gradient start) | `-600` (anchor) | Fill (10%) |
|---|---|---|---|
| Reading | `#22A68F` | `#1C7A6C` | `#E4F3F0` |
| Writing | `#8A6AE0` | `#6D4FC9` | `#EEEAFB` |
| Listening | `#4A90F0` | `#2C6FD1` | `#E7EFFC` |
| Speaking (coming soon) | `#D68556` | `#B8623D` | `#F5EAE3` |

Speaking should visually read as "not live yet" — use its color only at reduced opacity (~50%) with a "Coming soon" tag, per the PRD's roadmap status.

### 1.3 Signature accent — Band Gold

One accent is reserved for the thing that makes this a game, not just a grading tool: XP, levels, avatar frame unlocks, and the band score itself when it's the hero number on screen.

| Token | Hex | Use |
|---|---|---|
| `gold-600` | `#B8862E` | Band score display, XP bar fill, level-up state, unlocked frame ring |
| `gold-100` | `#F7EDD9` | Background fill behind XP/level widgets |

This is the one color allowed to feel a little celebratory. Everywhere else stays restrained so gold actually stands out when it appears.

### 1.4 Semantic

| Meaning | Token | Hex |
|---|---|---|
| Success / correct | `success-600` | `#2F8F5B` |
| Error / incorrect | `error-600` | `#C1443A` |
| Warning / pending | `warning-600` | `#B8862E` (shares gold — pending states aren't celebratory, but reuse avoids inventing a fifth accent family) |

### 1.5 Rules

- Max two accent colors visible in any single view: one skill color (contextual) + gold (if XP/level is shown). Never stack three skill colors on one screen.
- Text on a colored fill always uses that color's `-600` (or darker) shade — never black, never gray — so tags and badges don't go flat.
- Dark backgrounds (nav rail, modals-on-dark) use `ink-900`, not pure black.
- **v2.0:** skill color is now also allowed on `IconChip` gradients (§4b) — this is chrome that *identifies* content (a nav row into that skill), not an action button's own fill. Buttons themselves still never take a skill color (§5 still applies) — the rule was about actions, not identity markers, and `IconChip` is explicitly the latter.

---

## 2. Typography

Three typefaces, each with one job. Don't let an AI tool "helpfully" substitute a fourth.

| Role | Typeface | Where |
|---|---|---|
| Display | **Fraunces** (serif) | Landing/marketing headlines, level-up celebration modal, band-score reveal moment. Used sparingly — this is the one place the product allows itself some warmth and personality. |
| UI / body | **Inter** | Everything else: nav, forms, buttons, body copy, question text, table rows. This is a dashboard-heavy product; legibility at 13–14px matters more than character. |
| Data | **IBM Plex Mono** (tabular figures on) | Band scores, XP counts, countdown timers, submission IDs, dates in history tables. Numbers that need to line up in a column, or that carry the app's "precision" feel. |

### Type scale

| Token | Size / line-height | Weight | Use |
|---|---|---|---|
| `display-lg` | 40px / 1.1 | 600, Fraunces | Level-up modal, marketing hero |
| `display-sm` | 28px / 1.2 | 600, Fraunces | Band-score reveal on a graded result, auth page brand heading |
| `h1` | 24px / 1.3 | 600, Inter | Page title |
| `h2` | 19px / 1.35 | 600, Inter | Section header |
| `h3` | 16px / 1.4 | 600, Inter | Card header, question group title, nav-row title |
| `body` | 15px / 1.6 | 400, Inter | Default body copy |
| `body-sm` | 13px / 1.5 | 400, Inter | Secondary text, table cells |
| `label` | 12px / 1.4 | 500, Inter, uppercase, +0.04em tracking | Eyebrows like "READING · PASSAGE 2" |
| `data-lg` | 32px / 1 | 600, Plex Mono | Big band score |
| `data-sm` | 13px / 1 | 500, Plex Mono, tabular-nums | Inline XP/timer/ID |

---

## 3. Spacing, radius, elevation

- **Spacing unit:** 4px. Use the scale 4 / 8 / 12 / 16 / 24 / 32 / 48 / 64 — nothing off-scale.
- **Radius:** 8px for buttons/inputs/tags, 12px for cards, full-circle for avatar frames. Don't mix a third radius value in. (`IconChip` uses a fixed 14px "squircle" — a deliberate, singular exception documented here rather than silently introduced, since it's a distinct shape category — an icon container, not a card/button/tag.)
- **Elevation (v2.0 — revised from v1.0's "flat by default"):** cards carry a soft ambient shadow at rest (`shadow-sm`: barely-there, `0 1px 2px` + `0 1px 3px` at 5–6% opacity) — enough to lift them off the page without reading as a floating panel. Interactive cards (clickable nav rows, test cards) get a stronger lift on hover (`shadow-hover`: `0 12px 28px` at 14% opacity) plus a 2px translateY. The bigger `shadow-modal` (`0 8px 24px` at 12% opacity) stays reserved for things that truly float above content — modals, dropdowns. A resting screen should still read as calm; the shadow scale exists so *interaction* has somewhere to go, not so every surface fights for attention.

---

## 4a. Signature component — the Band Meter

This is the one recurring visual device the product should be remembered by. It's a segmented horizontal bar with 18 ticks (0 to 9 in 0.5 steps), filled up to the current value in gold. It replaces generic circular progress rings and plain percentage bars everywhere a "how far along / how good" concept appears:

- A graded submission's overall band (filled to that band, in `gold-600`)
- XP progress to next level (filled proportionally, same shape, relabeled)
- A reading/listening test's question-group progress while attempting

Same shape, three contexts — that repetition is what makes it feel designed rather than assembled.

```
[■■■■■■■■■■■■■□□□□□]  Band 6.5
0        4.5         9
```

Spec: 18 segments, 4px gap, 6px height, `hairline` background, `gold-600` fill, rounded 2px per segment. Label above in `data-sm`.

## 4b. Signature component — the Icon Chip (added v2.0)

A colored, iconified identity marker: a 14px-radius chip, sized ~40–48px, holding a simple 1.8px-stroke line icon. Two tones:

- **Skill tone** — background is a 135° gradient from that skill's `-500` to `-600`, icon in `surface-0` white, `shadow-sm` for a touch of lift. Used wherever a skill's identity needs to be conveyed *without repeating its name as adjacent text* — the canonical case is the Dashboard's practice-launch rows, where the row's own title already says "Luyện tập Reading"; a same-word tag next to it was pure redundancy. The icon replaces that tag.
- **Neutral tone** — `paper-50` background, hairline border, `ink-400` icon, no shadow. Used for non-skill actions that want the same row layout (history, logout, add) without borrowing skill color for something that isn't skill content.

Icon set is deliberately simple line icons (book/headphones/pen/mic for the four skills; clock/logout-arrow/plus for neutral actions) — no icon library dependency, just inline SVG paths, so the visual weight stays consistent with the rest of the system rather than importing a mismatched icon set's own opinions.

`SkillTag` (the plain text pill from v1.0) is **not** replaced by `IconChip` — they solve different problems. `SkillTag` still labels *content* (a test card's skill, a submission row's skill, a question group's skill) where there's no adjacent repeating text to worry about. `IconChip` is specifically for *navigation into* a skill, where the destination's own label already names it.

---

## 5. Core components (spec, not code)

- **Skill tag** — pill, 8px radius, skill's 10%-fill background, skill's `-600` text, `label` type. e.g. "READING" in teal. Used for content labeling (test cards, submission rows, question groups) — not for nav rows that already state the skill by name (use `IconChip` there instead, §4b).
- **Icon chip** — see §4b.
- **Status badge** — for `pending / submitted / graded / failed` (per the PRD's submission states): pending = `warning`, graded = `success`, failed = `error`, submitted = `ink-400` neutral. Never gold — gold is reserved for XP/level only.
- **Band score display** — `data-lg` number in `gold-600`, Band Meter beneath it, skill tag above it.
- **XP bar** — Band Meter shape, relabeled, with current level as a `label`-style eyebrow ("LEVEL 12") and XP-to-next as `data-sm` on the right.
- **Avatar frame** — circular avatar, frame is a 3px ring in `gold-600` if unlocked-and-equipped, `ink-400` outline if unlocked-but-unequipped, and simply absent (no ring) if locked — don't gray out a locked frame with a lock icon, that reads as punitive rather than aspirational.
- **Card** — `surface-0`, 12px radius, hairline border, `shadow-sm` at rest (v2.0). Interactive cards (nav rows, test cards — anything clickable) additionally get `shadow-hover` + a 2px lift on hover.
- **Question card** — a `Card` instance, skill tag top-left, question text in `body`.
- **Submission history row** — dense row (not a card), hairline row divider, skill tag, `data-sm` date, band score, status badge right-aligned.
- **Timer chip** — `data-sm` mono, `ink-900` on `paper-50`, ticks down; shifts to `error-600` text under 1 minute remaining. No background color change (stays calm) — the number itself carries the urgency.
- **Buttons** — primary: `gradient-primary` fill (a subtle dark navy diagonal gradient, not flat `ink-900` — v2.0), white text, `shadow-sm` at rest → `shadow-hover` + 1px lift on hover. Secondary: `surface-0` fill, hairline border, `ink-900` text. Only one primary button visible per screen. Skill colors never appear on a button's own fill/border/text — identity color belongs on the `IconChip` next to it, not the clickable surface itself (see §1.5's v2.0 note).

---

## 6. Voice, briefly

Since this is a grading product: errors and empty states should sound like an exam interface, not an apologetic assistant. "No submissions yet" + "Take your first test" (verb-first CTA), not "Oops, looks like you haven't done anything yet!" Corrections and feedback (Writing) stay in a neutral, specific register — say what's wrong and what to do, skip the encouragement-padding.

---

## 7. Best practices for vibecoding the frontend against this system

The failure mode with AI-generated frontends isn't bad taste in any one screen — it's that every screen quietly reinvents its own version of a button, a card, a spacing rule, because each prompt was answered in isolation. The fix is to make this document (or its machine-readable form) the thing every prompt points at, not the thing every prompt reinvents.

1. **Turn this doc into a tokens file first, before touching any screen.** A `design-tokens.css` (CSS custom properties) or `tailwind.config.js` `theme.extend` block with every color/type/spacing value above, named exactly as in the tables. Nothing gets built until this file exists.
2. **Lock the config so off-palette values are impossible, not just discouraged.** If you're on Tailwind, don't allow arbitrary values (`bg-[#123abc]`) — restrict to the extended theme. An AI assistant literally cannot invent a new blue if the only blues available are the ones you defined.
3. **Build the component library before the pages.** Ask for Button, Tag, StatusBadge, BandMeter, Card, AvatarFrame, IconChip as isolated components on one showcase/storybook-style page first. Only once those exist do you ask for "the submission history page" — composed from those pieces, not styled fresh.
4. **One component or one screen per prompt, never "restyle the whole app."** A prompt like "using only the tokens in design-tokens.css, rebuild the SubmissionRow component to match the spec in design-system.md §5 — don't introduce new colors, radii, or font sizes" gets a reviewable diff. "Make the app look more professional" gets a wall of unreviewable change.
5. **Give the assistant the constraint, not just the goal.** Instead of "make this look nicer," say "match §4a Band Meter spec exactly: 18 segments, 4px gap, gold-600 fill" — cite the section. Specific constraints produce specific, checkable output.
6. **Ask for a self-critique pass referencing the doc.** After a component is built: "check this against design-system.md — flag anything that doesn't match a token or introduces a new pattern." This catches drift before it ships.
7. **Review diffs for scope creep.** If a prompt to fix the timer chip also touched button padding somewhere else, reject that hunk. Vibe-coded regressions usually enter through "while I was in there."
8. **Keep a running notes file of decisions and rejected directions** (e.g. `frontend-notes.md`) — "tried a circular progress ring for band score, rejected in favor of the Band Meter because it ties to the actual 0.5-step grading scale"; "v1.0's flat cards read as plain once built, not clean — v2.0 added shadow-sm as the fix rather than reaching for a heavier neo-brutalist treatment again." This stops future sessions from re-litigating settled choices.
9. **Screenshot and compare, don't just read code.** If your tooling supports it, render the page and visually diff it against the spec — spacing and color drift is much easier to catch by eye than by reading className strings.
