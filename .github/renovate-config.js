// Global configuration for the self-hosted Renovate action. Repository
// configuration (extends, packageRules) lives in renovate.json at the root: a
// custom configurationFile is read as global config, so it is not the place for
// repository rules.
//
// There is deliberately no `username`. It is the account Renovate runs as, and
// it is what Renovate searches for when it looks for its own issues and
// pull requests - the dependency dashboard among them. Hardcoding
// "renovate[bot]" was wrong for this setup: without a RENOVATE_TOKEN the action
// runs as GITHUB_TOKEN, which is github-actions[bot], so Renovate never found
// the dashboard it had created and opened a new one on every run. Left unset,
// Renovate detects the account from the token in use.
module.exports = {
  gitAuthor: "Renovate Bot <bot@renovateapp.com>",
  onboarding: false,
  platform: "github",
  forkProcessing: "disabled",
  dryRun: null,
};
