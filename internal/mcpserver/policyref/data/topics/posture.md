# Posture

Use when:
- Adding device posture controls.
- Reasoning about `srcPosture` or `defaultSrcPosture`.
- Checking unset attribute behavior.

Rules:
- `postures` defines named posture expressions.
- `srcPosture` on grants or SSH rules requires source devices to satisfy named postures.
- `defaultSrcPosture` applies a default source posture to rules that do not define their own.
- `defaultSrcPosture` is replacing behavior, not additive with an explicit `srcPosture`.
- Operators compare built-in, custom, or integration posture attributes.
- Unset attributes generally do not satisfy comparisons unless the expression explicitly accounts for absence.
- Shared nodes and subnet-routed devices can bypass posture expectations in ways that matter for threat modeling.

Shape:
```json
{
  "postures": {
    "posture:trusted": ["node:os == 'macos'"]
  },
  "grants": [
    {
      "src": ["group:eng"],
      "dst": ["tag:prod"],
      "ip": ["tcp:443"],
      "srcPosture": ["posture:trusted"]
    }
  ]
}
```

Sources:
- https://tailscale.com/kb/1288/device-posture
- https://tailscale.com/kb/1337/policy-syntax
