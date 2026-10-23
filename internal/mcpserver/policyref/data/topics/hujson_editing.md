# HuJSON Editing

Use when:
- Editing raw policy text.
- Preserving comments and trailing commas.
- Understanding HuJSON limits.

Rules:
- Tailnet policy uses HuJSON/JWCC: JSON with comments and trailing commas.
- Line comments (`// ...`) and block comments (`/* ... */`) are valid.
- Trailing commas are valid in objects and arrays.
- Do not use JSON5-only syntax: unquoted keys, single-quoted strings, hexadecimal numbers, `Infinity`, `NaN`, or plus-prefixed numbers.
- Make minimal textual edits when possible so existing comments and layout survive.
- API validation or normalization may preserve valid policy behavior while changing formatting or comments. Do not depend on comment round-tripping after a server-side normalize/save path.

Practical edits:
- Keep added entries near related entries.
- Preserve existing indentation style.
- Prefer appending a new rule over rewriting an entire section.
- Run policy evaluation before staging.

Live Markdown: https://tailscale.com/docs/features/tailnet-policy-file.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/features/tailnet-policy-file
- https://tailscale.com/docs/reference/syntax/policy-file
- https://github.com/tailscale/hujson
- https://nigeltao.github.io/blog/2021/json-with-commas-comments.html
