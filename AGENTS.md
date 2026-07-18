# Repository Agent Rules

## GitFlow

Use GitFlow for every code or configuration change in this repository.

### Long-lived branches

- `main` contains production-ready code only. Never commit directly to it.
- `develop` is the integration branch for the next release. Never commit directly
  to it.
- Changes reach `main` and `develop` through reviewed pull requests or merges only.

### Working branches

- Create `feature/<short-kebab-name>` from `develop` for features, maintenance,
  documentation, and non-urgent fixes. Merge it back into `develop`.
- Create `release/<version>` from `develop` for release stabilization. Only release
  fixes and version or documentation updates are allowed. Merge it into both
  `main` and `develop`, then tag `main` with `v<version>`.
- Create `hotfix/<short-kebab-name>` from `main` for urgent production fixes.
  Merge it into both `main` and `develop`, and tag the resulting release when
  appropriate.
- Delete working branches after they have been merged.

### Agent workflow

1. Before editing, inspect `git status` and the current branch. Preserve all
   pre-existing user changes.
2. For an authorized implementation, work on the correctly based GitFlow branch.
   If the requested branch operation is ambiguous or could disrupt local changes,
   ask before creating or switching branches.
3. Keep commits focused and use Conventional Commits, for example
   `feat: add peer routing`, `fix: correct dashboard port`, or
   `docs: update deployment steps`.
4. Run relevant validation before proposing a merge and report anything that was
   not tested.
5. Do not merge, tag, push, force-push, rebase shared branches, or delete remote
   branches unless the user explicitly requests that action.
6. Never bypass branch protection, rewrite published history, or commit secrets,
   credentials, generated runtime data, or local environment files.

