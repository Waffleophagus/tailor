## Project

This is tailor, it has a simple goal: to map out a tailscale tailnet and allow you to visualize changes to your ACL policies. The goal is to have as simple an interface as possible to simplify the ability for the user to, at a glance, see how their tailnet is structured and what devices/users can talk to.


## Skill

The `impeccable` skill (from `skills-lock.json`) is available for UI work. Use it, abuse it, the skills are invaluable.

## Issue tracker

Issues are local markdown files under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

Policy editing and simulation roadmap: `.scratch/tailnet-topology-visualizer/018-policy-scenario-roadmap.md` — start with [019-policy-workbench-shell.md](.scratch/tailnet-topology-visualizer/019-policy-workbench-shell.md). Reference UI screenshots: `screenshots of ACL in site/`.

### Triage labels

`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.


## Rules
When you are done modifying anything in the front end, run the following in order
```
pnpm format && pnpm lint
```
then fix issues, and then run
```
pnpm check
```
and make sure you have not introduced issues.