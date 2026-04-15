# Charm Docs

Technical writing for terminal UI work. Show, don't tell.

## Prose

- Friendly and clear — not formal, not slangy
- Lead with a visual example or code snippet, then explain what it does
- Show don't tell: a 5-line code block beats a paragraph of description
- Short paragraphs — rarely more than 3 sentences
- When describing a visual element, say what it looks like:
  "a rounded border in gold with 1-cell padding on each side"

## Code Comments

- Explain the visual intent, not just the mechanics:
  `// gold header bar spanning full terminal width`
  not `// set background color and width`
- `// NarrationPane renders the DM's narration text with soft wrapping`
  not `// renders text`
- Group related style definitions with a section comment:
  `// -- Combat overlay styles --`
- Inline comments for non-obvious layout math:
  `// subtract sidebar (30) + border (2) + padding (2) from terminal width`

## Naming

- Component names describe what the user sees:
  `NarrationPane`, `CombatOverlay`, `StatsSidebar`, `InventoryList`,
  `MiniMap`, `PromptBar`
- Style variable names describe appearance:
  `headerStyle`, `goldAccent`, `dimText`, `activeBorder`,
  `selectedRow`, `mutedForeground`
- Layout constants describe their purpose:
  `sidebarWidth`, `minContentWidth`, `headerHeight`, `statusBarHeight`
- Message types describe the event:
  `CombatStartedMsg`, `NarrationReadyMsg`, `PaneResizedMsg`

## Documentation

- Lead with what it looks like — a screenshot, an ASCII rendering,
  or a code example that produces visible output
- Then explain the API: what functions to call, what options exist
- End with edge cases: what happens on narrow terminals, with long
  text, with no color support
- README examples should be copy-pasteable and produce visible results

## Error Messages

- Include what the user will see:
  `rendering narration pane: terminal width %d too narrow for layout (minimum %d)`
- Never bare `return err` — always add rendering context
