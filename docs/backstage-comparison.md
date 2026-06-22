# Spire manifest vs. Backstage Scaffolder template

A comparison of the Spire manifest model (`.spire/manifest.yaml`) against Backstage's
Scaffolder template format (`template.yaml`), to assess what it would take to generate
Backstage-compatible templates from Spire.

## The two models at a glance

| | **Spire** (`.spire/manifest.yaml`) | **Backstage** (`template.yaml`) |
|---|---|---|
| Kind | bespoke `SpireManifest` struct | `kind: Template`, `apiVersion: scaffolder.backstage.io/v1beta3` |
| Where it lives | inside the template repo **and** travels into the generated project | in the template repo only (registered in the catalog); never copied to output |
| Lifecycle | `init` → `service add` → `upgrade` (stateful, re-runnable) | one-shot `scaffold` run (stateless, fire-and-forget) |
| Input collection | `appSlots` (flat list, prompted in CLI) | `spec.parameters` (JSON Schema, rendered as a web form) |
| Templating engine | Go `text/template`, custom `[[ ]]` delimiters | Nunjucks, `${{ }}` delimiters |
| Rendering trigger | any file containing `[[ ]]` is rendered | `fetch:template` action over a `skeleton/` dir |
| Output | files on disk in the user's cwd | a published Git repo + catalog registration |
| Runtime | local CLI, interactive prompts | server-side action pipeline, no interactivity |

## Field-by-field mapping

| Spire concept | Backstage equivalent | Fit |
|---|---|---|
| `appSlots[].key` | `parameters.properties.<key>` | ✅ direct |
| `appSlots[].label` | `property.title` | ✅ |
| `appSlots[].description` | `property.description` | ✅ |
| `appSlots[].defaultValue` | `property.default` | ✅ |
| `type: PromptMandatory` | listed in `required: []` | ✅ |
| `type: PromptOptional` | not in `required` | ✅ |
| `type: PromptSecret` | `ui:field: Secret` / `ui:widget: password` | ⚠️ partial — Backstage secrets are a different mechanism |
| `type: DynamicValue` + `expression` | **no equivalent** in parameters; must move into a `steps` templating step | ❌ conceptual gap |
| `validation` (string rules) | JSON Schema keywords (`pattern`, `minLength`, `enum`, `maximum`…) | ⚠️ needs a translator (see below) |
| `pipelines` (`slugify`, `camelCase`…) | Nunjucks filters in steps (`parseEntityRef`, custom filters) | ⚠️ partial — not all map |
| `templateFiles` | `fetch:template` step + `skeleton/` layout | ⚠️ different model |
| `pathRenames` | done via templated file/dir names in skeleton (`${{ values.x }}` in path) | ⚠️ different model |
| `ignorePaths` | `fetch:template` `copyWithoutTemplating` / cookiecutterCompat | ⚠️ partial |
| `services` / `serviceConfig` / `service add` | **no equivalent** — Backstage is one-shot, no incremental add | ❌ no concept |
| `postHooks` (conditional removePaths) | `if:` conditions on steps, or `roadiehq:utils:fs:delete` | ⚠️ partial |
| `upgrade-manifest.yaml` | **no equivalent** — Backstage doesn't re-run/upgrade | ❌ no concept |
| (implicit: write to cwd) | `publish:github` + `catalog:register` steps | ❌ Spire has no publish step |

## The four real gaps (in priority order)

**1. Validation translation (easy).** The `validation` strings are already a near 1:1
with JSON Schema. `minLength:2` → `minLength: 2`, `enum:yes,no` → `enum: [yes, no]`,
`port` → `maximum: 65535, minimum: 1`, `slug`/`email`/`semver` → `pattern: <regex>`. A
small lookup table covers all 18 rules.

**2. Templating delimiter + filter rewrite (medium).** Every rendered file would need
`[[ .slots.X ]]` → `${{ values.X }}` and `[[ .slots.X | slugify ]]` →
`${{ values.X | <filter> }}`. Backstage doesn't ship `slugify`/`pascalCase`/`snakeCase`
out of the box — we'd either restrict the pipeline set we support, register custom
Nunjucks filters in the Backstage instance, or pre-compute these as derived parameters.
This is the biggest mechanical lift because it touches file contents, not just the manifest.

**3. `DynamicValue` slots (medium).** These have no home in `parameters` (the form layer).
The clean translation is a templating step that derives them, or precomputing them into
the skeleton — either way they leave the manifest and become part of the steps pipeline.

**4. Steps / publish pipeline (we'd author this).** Backstage needs `spec.steps` we don't
currently model at all: `fetch:template` → `publish:github` → `catalog:register`. This is
boilerplate we'd template, but `services`, `serviceConfig`, `postHooks`, and the whole
`upgrade` story have **no Backstage counterpart** — Backstage scaffolds once and walks
away. Those features simply don't export.

## Bottom line on "how far are we"

For a **single-shot project skeleton** (appSlots + templateFiles + pathRenames +
ignorePaths), we're **close** — roughly a manifest transformer plus a delimiter/filter
rewriter:

- `appSlots` → `parameters` JSON Schema (mostly mechanical) ✅
- `validation` → JSON Schema keywords (lookup table) ✅
- emit a standard 3-step `fetch/publish/register` pipeline ✅
- rewrite `[[ ]]` → `${{ }}` and resolve `DynamicValue`/pipelines ⚠️ the actual work

What **does not cross over** is everything that makes Spire stateful: `services` /
`service add`, `serviceConfig`, `postHooks`, and `upgrade`. Backstage has no notion of
adding to or upgrading an already-scaffolded project, so those features are out of scope
for export by design — not a thing to "catch up" on.

## Suggested first deliverable

A `spire template export --backstage` command that emits `template.yaml` + a `skeleton/`
tree, supporting the one-shot subset and explicitly warning when a template uses
`services`/`upgrade` features that can't be represented.
