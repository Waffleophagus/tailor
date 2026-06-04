# Tests

Use when:
- Adding validation tests to policy drafts.
- Proving intended allow or deny behavior.
- Testing SSH or posture-dependent access.

Network tests:
- `tests` verifies expected allowed and denied network access.
- Include `src`, `accept`, and/or `deny` expectations.
- ICMP tests use ICMP-specific destination handling.
- Test destinations are restricted and cannot use CIDRs.
- Use tests when changing grants, ACLs, groups, tags, hosts, or ipsets.

SSH tests:
- `sshTests` verifies SSH rule behavior.
- Include source, destination, user, and expected accept/check/deny behavior.

Posture tests:
- `srcPostureAttrs` can provide source posture attributes for test evaluation.
- Use posture tests when adding or changing `srcPosture` or `defaultSrcPosture`.

Common mistakes:
- Testing only allow behavior and not the intended deny boundary.
- Using CIDR destinations in tests.
- Forgetting that grants are additive, so an unrelated broader rule may make a deny test fail.

Sources:
- https://tailscale.com/kb/1337/policy-syntax
- https://tailscale.com/kb/1018/acls
