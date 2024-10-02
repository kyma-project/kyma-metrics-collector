# Releasing

## Release Process

This release process covers the steps to release new major and minor versions for the Kyma Metrics Collector.

1. Trigger a new release by manually triggering the "Create release" Github action and provide a proper release tag as argument, like "1.2.0". The action will perform the following steps:
  - Verify the tag
  - Verify that all PRs have proper labeling
  - Execute unit tests
  - Create PR to bump versions in security config
  - Wait for the merge of the PR
  - Create a draft release
  - Create the tag
  - Wait for release artifact creation
  - Publish the release
