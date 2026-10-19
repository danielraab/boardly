# CLAUDE.md

See [README.md](README.md) for what the project is and how to run it.

## Workflow: always use OpenSpec

Every code change goes through [OpenSpec](openspec/). Do not edit code under `backend/` or `frontend/` without an OpenSpec change.

1. **Explore** (optional): `/opsx:explore` to think through an idea or clarify requirements.
2. **Propose**: `/opsx:propose` to create a change in `openspec/changes/<name>/` with proposal, design, specs and tasks.
3. **Apply**: `/opsx:apply` to implement the change's tasks and tick them off as you go.
4. **Archive**: `/opsx:archive` when the change is done, which syncs its delta specs into `openspec/specs/`.

If a request asks for a code change and no matching OpenSpec change exists, propose one first.
If the work turns out to differ from the change, update the change's artifacts before changing the code.

Exceptions that don't need a change: typo fixes, formatting or lint-only fixes, dependency bumps, and documentation-only edits.
