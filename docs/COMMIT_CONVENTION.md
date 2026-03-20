# Commit Message Convention

Format:

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

`scope` is optional. When provided, it should identify the module or layer affected.

---

## Type

| Type | When to use |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `refactor` | Code restructuring with no behavior change |
| `test` | Adding or updating tests |
| `docs` | Documentation only changes |
| `chore` | Maintenance tasks (dependencies, config, tooling) |
| `perf` | Performance improvement |
| `style` | Formatting, whitespace, missing semicolons — no logic change |
| `revert` | Reverting a previous commit |
| `ci` | CI/CD pipeline changes |

---

## Scope (optional)

Enclose in parentheses after the type. Use the package or layer name:

| Scope | Covers |
|-------|--------|
| `proxy` | `internal/proxy` and sub-packages |
| `auth` | `internal/auth` |
| `quota` | `internal/quota` |
| `chat` | `internal/chat` |
| `model` | `internal/model` |
| `config` | `internal/config` |
| `db` | `internal/db`, migrations |
| `middleware` | `internal/middleware` |
| `providers` | `internal/proxy/providers` |
| `web` | `web/` frontend |
| `deps` | Dependency updates (`go.mod`, `package.json`) |

---

## Subject

- Imperative mood: "add", "fix", "remove" — not "added" / "fixes"
- No capital first letter
- No period at the end
- Max ~72 characters

---

## Examples

```
feat(auth): add WeChat Work SSO provider

fix(proxy): save ChatLog in non-streaming path

fix(handler): use errors.Is for ErrQuotaExceeded comparison

refactor(quota): extract QuotaService interface for testability

test(proxy): add unit and integration tests for handler and router

docs: add API reference and architecture overview

chore(deps): upgrade gin to v1.10.0

perf(proxy): reuse http.Transport across provider instances

revert(auth): revert JWT expiry change pending security review
```

---

## Body (optional)

Use the body to explain **why**, not what. Wrap at 72 characters.

```
fix(proxy): save ChatLog in non-streaming path

Non-streaming requests were only deducting quota but never persisting
the ChatLog, making audit logs incomplete. Streaming path was correct
because providers handle saving internally.
```

---

## Footer (optional)

- Reference issues: `Closes #42`, `Refs #17`
- Note breaking changes: `BREAKING CHANGE: <description>`

```
feat(quota): change token deduction to use atomic update

BREAKING CHANGE: user_quotas table requires a new index on (user_id, model_id).
Run db/migrations/005_add_quota_index.sql before deploying.

Closes #38
```
