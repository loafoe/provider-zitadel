// Global configuration for the self-hosted Renovate action. Repository
// configuration (extends, packageRules) lives in renovate.json at the root: a
// custom configurationFile is read as global config, so it is not the place for
// repository rules.
module.exports = {
  username: "renovate[bot]",
  gitAuthor: "Renovate Bot <bot@renovateapp.com>",
  onboarding: false,
  platform: "github",
  forkProcessing: "disabled",
  dryRun: null,
};
