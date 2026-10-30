module.exports = {
  username: "renovate[bot]",
  gitAuthor: "Renovate Bot <bot@renovateapp.com>",
  onboarding: false,
  platform: "github",
  forkProcessing: "disabled",
  dryRun: null,
  extends: ["config:recommended", ":semanticCommitsDisabled"],
  packageRules: [
    {
      matchManagers: ["gomod", "github-actions"],
      groupName: "Go and CI dependencies",
    },
  ],
};
