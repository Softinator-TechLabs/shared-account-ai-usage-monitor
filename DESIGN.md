---
name: Shared Account AI Usage Monitor
description: A graphite, cool-white and cobalt workspace for people and recorded conversations.
colors:
  paper: "#f8f9fc"
  surface: "#fff"
  ink: "#20232d"
  muted: "#666c7d"
  line: "#e5e7ef"
  action: "#305dd9"
  action-hover: "#244cb9"
  wash: "#f0f4ff"
  rail: "#181c27"
  rail-text: "#aeb7ca"
  rail-hover: "#242b3a"
  rail-selected: "#2e3c64"
  rail-selected-text: "#f2f5ff"
  control-border: "#ccd1dd"
  control-hover: "#f4f6fb"
  control-hover-border: "#abb6d1"
  focus: "#6586e9"
  warning: "#685323"
  warning-bg: "#fbf7ed"
  role-bg: "#f0f2f7"
  role-text: "#56627b"
typography:
  headline:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "36px"
    fontWeight: 550
    lineHeight: 1.15
    letterSpacing: "-0.035em"
  title:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "21px"
    fontWeight: 550
    lineHeight: 1.3
    letterSpacing: "-0.025em"
  subtitle:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "16px"
    fontWeight: 600
  body:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.65
  conversation:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.65
  label:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "13px"
    fontWeight: 600
  field:
    fontFamily: "Geist, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 400
rounded:
  tag: "4px"
  control: "8px"
  surface: "12px"
  workspace: "16px"
spacing:
  tight: "6px"
  small: "8px"
  compact: "12px"
  regular: "16px"
  row: "20px"
  section: "24px"
  panel: "28px"
  heading: "32px"
components:
  button-primary:
    backgroundColor: "{colors.action}"
    textColor: "{colors.surface}"
    rounded: "{rounded.control}"
    padding: "10px 14px"
  button-primary-hover:
    backgroundColor: "{colors.action-hover}"
    textColor: "{colors.surface}"
  button-secondary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "10px 14px"
  button-text:
    backgroundColor: "transparent"
    textColor: "{colors.action}"
    padding: "5px 0"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.field}"
    rounded: "{rounded.control}"
    padding: "10px 14px"
  navigation-item:
    backgroundColor: "transparent"
    textColor: "{colors.rail-text}"
    rounded: "{rounded.control}"
    padding: "12px"
  navigation-current:
    backgroundColor: "{colors.rail-selected}"
    textColor: "{colors.rail-selected-text}"
  role-tag:
    backgroundColor: "{colors.role-bg}"
    textColor: "{colors.role-text}"
    rounded: "{rounded.tag}"
    padding: "3px 5px"
  inline-panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.surface}"
    padding: "28px"
  session-row:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    padding: "20px"
  session-row-selected:
    backgroundColor: "{colors.wash}"
    textColor: "{colors.ink}"
---

# Design System: Shared Account AI Usage Monitor

## Overview

**Creative North Star: "People and sessions"**

This is a source-derived record of the implemented proposal, not a user-approved brand system. The user rejected the earlier green palette and typography. The replacement graphite, cool-white, cobalt and Geist direction was selected by the agent; approval remains pending. Tokens describe this build and should preserve its consistency while it is evaluated.

The workspace gives people, device facts and recorded conversation content a quiet reading surface. A dark navigation ground frames white work areas; restrained blue actions and selected states guide movement. Structured rows, thin rules and readable conversation text carry the hierarchy. The north-star wording is descriptive and comes from the direction contract, not a newly confirmed metaphor.

**Key Characteristics:**

- Graphite navigation with cool-white content surfaces.
- Self-hosted Geist throughout headings, controls and conversation prose.
- Row-based records and thin separators, with flat surfaces at rest.
- Cobalt actions and soft blue selection states.
- Responsive navigation and explicit labels for identity, devices and account facts.

Source: `internal/web/assets/app.css`, `index.html` and `app.js`, inspected 2026-09-26. This documentation records implemented styling; it does not establish deployment, product acceptance or employee-device verification.

## Colors

The palette combines a deep graphite frame with cool near-white surfaces and a clear cobalt action color.

### Primary

- **Cobalt action** (`action`): primary buttons, text actions, links, carets and review scores. Its deeper hover color gives immediate pointer feedback.
- **Blue wash** (`wash`): selected session rows, paired with an inset selection boundary.
- **Focus blue** (`focus`): visible keyboard outlines around interactive controls.

### Neutral

- **Cool paper** (`paper`): page background and expanded source-field backgrounds.
- **White surface** (`surface`): forms, roster, reader and account containers.
- **Graphite rail** (`rail`): navigation ground; muted rail text brightens on interaction and selection.
- **Ink** (`ink`): headings, names and conversation content.
- **Muted slate** (`muted`): explanatory copy, timestamps and supporting facts.
- **Soft rule** (`line`): rows, panel edges and section boundaries.
- **Control border** (`control-border`): fields and secondary buttons, with a stronger hover boundary.
- **Role slate** (`role-bg`, `role-text`): compact factual role labels.

Warm warning colors distinguish explanatory notices. Green appears only in the implemented connected-device status indicator; the rejected green brand direction is not a prohibition on factual status color.

**The Action and Evidence Rule.** Use cobalt for actions and selection; keep long evidence content in ink on a light surface.

## Typography

**Display and Body Font:** Self-hosted Geist, with system-ui and sans-serif fallbacks. The variable font loads from `/geist.woff2` with swap behavior; system faces are fallbacks, not the intended display identity.

The hierarchy uses restrained weight changes and slightly tightened headings. Conversation prose retains whitespace and wraps long strings rather than shrinking to fit.

### Hierarchy

- **Headline:** the headline token for page titles; mobile page headings reduce to 30px. The login headline is a local 38px/500 variation, reducing to 32px on narrower layouts.
- **Title:** the title token for section headings; list headings use the subtitle size.
- **Subtitle:** compact section names and supporting headings.
- **Body:** normal explanatory copy and workspace text.
- **Conversation:** the conversation token for full message content, constrained to 70ch. Expanded source data uses the browser's monospace treatment separately.
- **Label and field:** the label and field tokens for forms. Mobile fields use 16px text and a minimum 44px height.

**The Reading Width Rule.** Preserve message whitespace, wrap long content and keep the primary conversation measure at 70ch.

## Layout

Desktop uses a fixed, independently scrolling navigation rail at the left and a document-scrolling work area. The standard rail is 224px wide and 100dvh high. Main content, feedback, policy strip and footer share its offset. The page background continues the navigation ground down long documents. Main content has a maximum width of 1800px with 40px top, 38px side and 64px bottom padding.

At 1200px and below, the header and content offset become 200px, the main gutter becomes 24px and the session list narrows from 320px to 250px. The document background follows the same 200px rail. At 960px and below, the login image is removed and its form becomes a single-column surface; the separate account-column omission at intermediate widths is not an endorsed information rule.

At 760px and below, navigation becomes an in-flow full-width top region with a horizontal row of icon-over-label controls. The desktop offsets disappear, the paper background fills the page and main gutters become 16px. The directory becomes stacked two-column records: identity spans the width, empty device/account facts can share a row, and records with devices let device details span the width. Session counts and available actions occupy the lower grid cells and wrap as needed. Search and review forms stack; the session list moves above the reader and scrolls within a maximum 310px height.

The spacing scale reflects repeated source values rather than a mechanically uniform grid. Preserve larger separation between sections and tighter spacing inside factual groups. The Add person form opens inline immediately after the heading, ahead of the roster.

## Elevation & Depth

Surfaces are flat at rest. Background tones, thin borders and whitespace establish depth; there are no ambient or offset drop shadows. The selected session row uses an inset 1px boundary (`inset 0 0 0 1px #b8caf8`), which is a state cue, not elevation. Keyboard focus uses a 3px outline with a 3px offset.

**The Flat Surface Rule.** Use tone and rules to separate work areas; preserve the selected row's inset boundary as a selection cue.

## Shapes

Controls use softly rounded corners, containers use broader rounding, and small role labels remain compact rectangles. Follow the named radius tokens for reusable primitives. The workspace and login use the largest shared radius. Avatar tiles use softly rounded squares; connected-state dots alone are circular. Full-width list rows have square internal corners so their separators remain continuous.

Icons are inline stroked SVG with rounded joins and caps, usually 18px in controls. They accompany text; they do not replace readable action labels. The authored shared-workspace raster supports the login and connection note, without becoming a background for primary evidence.

## Components

### Buttons

Compact controls with explicit action text. Primary buttons use cobalt and white, secondary buttons use white with a neutral border, and text actions use cobalt without a filled background. Default padding and corner shape are tokenized above. Primary hover darkens; secondary hover changes fill and border; text action hover underlines. Buttons transition background and border over 160ms with ease. Disabled buttons use reduced opacity and a waiting cursor. Focus remains visible across variants.

### Inputs / Fields

White fields with neutral strokes and control rounding. Labels sit above their controls with a 6px gap. Inputs, selects and textareas share form typography; textareas allow vertical resizing with a 120px minimum height. Search uses a visually hidden accessible label where space is tight. Preserve the mobile field size and visible focus outline.

### Chips

Role labels use a small neutral fill and capitalized text. Account assignments use thin bordered tags that wrap. Counts use small neutral rounded tiles. These are factual annotations, not elevated controls or decorative eyebrow headings.

### Cards / Containers

The roster is a single bordered white container with internal row rules. Inline forms share white surfaces and surface rounding; the Add person panel has a 560px maximum width. The conversation workspace is one divided surface, with a tinted list and white reader. Avoid treating every record or message as a separate floating card.

### Navigation

The graphite navigation uses icon-and-label buttons, a darker hover surface and a blue-gray current-page fill. Active state is represented with `aria-current`. On desktop it is vertical; mobile places the icon above the label with a 52px minimum item height. The identity and sign-out controls remain grouped within the navigation region.

### Session rows and conversation

Each session row contains a strong title, a three-line preview and muted client/context metadata below the preview. A blue wash and inset boundary distinguish the current selection. Messages use top rules and spacious vertical padding; the discussion action points to one message, whose selected state receives a very pale background and outline. Full content remains readable prose, not monospace merely because the HTML element preserves whitespace.

### Inline form motion

The Add person panel unfolds over 320ms using `cubic-bezier(0.16, 1, 0.3, 1)`, with a small upward starting offset and clipping. Closing or successful submission returns focus to its opener. Reduced-motion preferences remove transitions and animations. Do not extend this one purposeful reveal into continuous ornament.

## Do's and Don'ts

### Do:

- **Do** use the implemented graphite, cool-white and cobalt palette consistently while the proposal is under evaluation.
- **Do** preserve visible focus, accessible text labels and responsive field sizing.
- **Do** keep long evidence readable, with wrapping and preserved whitespace.
- **Do** distinguish factual metadata, action affordances and selected states through their established treatments.
- **Do** use thin rules and tonal surfaces to group content.

### Don't:

- **Don't** present this agent-selected replacement as user-approved.
- **Don't** reintroduce the rejected green brand palette and typography.
- **Don't** promote decorative eyebrow headings, colored card edges or hard offset shadows into this system.
- **Don't** inherit breakpoint omissions as design conventions.
- **Don't** let supporting imagery compete with the primary evidence-reading surface.
