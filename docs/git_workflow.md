# LinkPulse Engineering Workflow & Conventions

This document outlines the source control workflow, commit guidelines, and version release lifecycle conventions followed in the development of **LinkPulse**.

---

## 1. Git Branching Strategy (Git Flow Lite)

We follow a structured branching system based on **Git Flow** simplified for high-velocity continuous integration:

```
                  ┌──────────────────────────────────────────────┐
                  │                 main (prod)                  │
                  └──────────────────────▲───────────────────────┘
                                         │ Merge PR
                  ┌──────────────────────┴───────────────────────┐
                  │               develop (staging)              │
                  └──────────────────────▲───────────────────────┘
                                         │ Merge PR
                    ┌────────────────────┴────────────────────┐
                    │                                         │
       ┌────────────┴────────────┐               ┌────────────┴────────────┐
       │     feature/url-stats   │               │     bugfix/cache-miss   │
       └─────────────────────────┘               └─────────────────────────┘
```

### Branch Names
- **`main`**: Production-ready code only. Direct commits are forbidden. Every merge is tagged with a release version (e.g. `v1.0.0`).
- **`develop`**: Integration branch for features. Staged to pre-production environments automatically.
- **`feature/<name>`**: Short-lived branches created for single issues or feature cards. Merged into `develop` via Pull Request review.
- **`bugfix/<name>`**: Used to fix bugs discovered in develop or pre-prod.
- **`hotfix/<name>`**: Created directly from `main` to address critical production issues. Merged to both `main` and `develop`.

---

## 2. Commit Message Conventions (Conventional Commits)

Commit messages must be descriptive and follow the **Conventional Commits** specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Commit Types
- **`feat`**: A new feature (e.g. `feat(api): add custom slug validations`)
- **`fix`**: A bug fix (e.g. `fix(cache): resolve redis key collision`)
- **`docs`**: Documentation changes (e.g. `docs(readme): add system diagrams`)
- **`style`**: Changes that do not affect the meaning of the code (formatting, missing semi-colons, etc.)
- **`refactor`**: Code changes that neither fix a bug nor add a feature
- **`perf`**: A code change that improves performance
- **`test`**: Adding missing tests or correcting existing tests
- **`chore`**: Maintenance tasks or build process updates

### Example
```text
feat(service): implement max retries for short code generator

Read the retry count from configurations instead of hardcoding 5.
Retries limit ensures generation halts if keyspace is saturated.

Closes #42
```

---

## 3. Release & Versioning Approach (Semantic Versioning)

LinkPulse follows **Semantic Versioning (SemVer)**: `MAJOR.MINOR.PATCH`.

- **`MAJOR`**: Incompatible API changes (e.g., changing response structure, breaking endpoint routes).
- **`MINOR`**: Adding functionality in a backwards-compatible manner (e.g. adding analytics collection, registration endpoints).
- **`PATCH`**: Backwards-compatible bug fixes (e.g., repairing GORM connection pools leak, updating dependencies).

### Release Lifecycle
1. Compile the changelog for the milestone.
2. Build the Docker container tagged with the version (e.g. `linkpulse:1.2.0`).
3. Create a Git tag:
   ```bash
   git tag -a v1.2.0 -m "Release v1.2.0"
   git push origin v1.2.0
   ```
4. Deploy version artifact to production environments.
