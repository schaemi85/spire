# Upgrade from Scaffolding Commands

This directory contains all commands related to upgrading a Spire-based project from its scaffolding template. These commands form a cohesive system for safely synchronizing projects with upstream scaffolding changes.

## Usage Examples

A typical upgrade workflow can be run as a sequence of commands (manually or in a script/task):

```bash
# 1. Execute upgrate to the latest available template
spire upgrade [--dry-run]

# 3. Proceed with the manual review of the upgrade and adjust as needed the code
# 4. Commit & push changes
```

## Upgrade process

Lets use a Replace model with explicit exceptions (ie a pattern for specific files)
We backup files to keep (listed here and prefixed one), delete entirely the folder (to ensure we remove useless files)
We regenerate the one that are generated.
For the config files we only provide example to help the user to update it manually to the new setup.

Deleting then regenerating avoids “template drift” and dead files.
Explicit keep rules (prefix-based or path-based) are safer than trying to merge everything.

### Files and folders rules

.devcontainer: all except:

- .devcontainer/postgres/init-user-db.sh
- .devcontainer/.env

.vscode: all except this to regenerate:

- .vscode/launch.json

infra/tofu/azure : all except additional custom files to be kept and generated files to be regenerated

- infra/tofu/azure/environments

infra/tofu/tufin : all except additional custom files to be kept

- infra/tofu/tufin/environments

pkg : all except custom one

templates/ all folder but be careful of the submodule folder

.dockerignore
.gitattributes
.gitignore
.gitlab-ci.cloud.yml
.gitlab-ci.onprem.yml
.gitlab-ci.yml
.golangci.yml
Taskfile.yml

services replacements:

- all infrastructure modules

Once all file overwritten we have to apply the same process as when initializing new project

Best approach use an upgrade manifest:

longest path wins to avoid edge cases

```yaml
version: 1
application:
  - path: .devcontainer
    keep:
      - postgres/init-user-db.sh
      - .env

  - path: .vscode

  - path: infra/tofu/azure
    keep:
      - environments/**
      - "**/custom_*.tf"

  - path: infra/tofu/tufin
    keep:
      - environments/**
      - "**/custom_*.tf"

  - path: pkg

  - path: templates
  - path: .dockerignore
  - path: .gitattributes
  - path: .gitignore
  - path: .gitlab-ci.cloud.yml
  - path: .gitlab-ci.onprem.yml
  - path: .gitlab-ci.yml
  - path: .golangci.yml
  - path: Taskfile.yml

services:
  - path: _apigw
  - path: _azure
  - path: _k8s

```

### Upgrade process steps

Refuse to upgrade if git status is dirty (unless --force)

Backup:

- for each application/services paths: Copy all kept files to a `.spire/backup/<timestamp>/` Or a temp dir if --dry-run

Reconcile:

- For each application path:
  - Delete target path
  - Copy path from template
  - Restore kept files (overwrite if already present)

- Then do the same for each services path

PostInit hooks:
search and replace placeholders and templating
go mod tidy
formatting / lint config

### Users steps

1. Upgrade Spire CLI: go install github.com/schaemi85/spire@v1.8.1
1. Proceed with CLI automated process
1. Review manually and adjust as needed
1. Tests : go mod tidy / go build
1. Commit and push changes accross environments and proceed with all check

To revert upgrade, just revert changes in git.
