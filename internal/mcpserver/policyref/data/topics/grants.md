# Grants

Use when:
- Adding or changing modern network access rules.
- Adding application capabilities.
- Routing traffic through selected tagged routers with `via`.

Rules:
- A grant requires `src` and `dst`.
- Include at least one of `ip` or `app`.
- `src` is a list of source selectors.
- `dst` is a list of destination selectors.
- `ip` grants network access. Use protocol/port entries such as `"tcp:443"`, `"udp:53"`, `"icmp"`, or `"*"` depending on intent.
- `app` grants application capabilities. App capabilities do not provide useful access unless matching network access also exists.
- `via` restricts the route path through tagged devices and accepts only tags.
- Grants are additive and unioned.
- `srcPosture` can require posture checks for sources.

Shape:
```json
{
  "src": ["group:eng"],
  "dst": ["tag:web"],
  "ip": ["tcp:443"]
}
```

Common mistakes:
- Creating an app-only grant without network access.
- Expecting a narrower grant to override a broader one.
- Using a non-tag selector in `via`.
- Treating a CIDR destination grant as route advertisement or route injection.

Sources:
- https://tailscale.com/kb/1324/grants
- https://tailscale.com/kb/1337/policy-syntax
