#!/usr/bin/env bash
# move changes to the dedicated branch created from the remote main

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables passed (passed from CI):
#   GH_TOKEN                      - GitHub token for GitHub CLI
#   GIT_EMAIL                     - email setting for PR to be created
#   GIT_NAME                      - user name setting for PR to be created
#   KYMA_METRICS_COLLECTOR_REPO   - KMC repository
#   BUMP_KMC_BRANCH_NAME          - branch with changes

TAG=$1

# add changed files to stage
git add resources/kyma-metrics-collector/values.yaml
git add resources/kyma-metrics-collector/Chart.yaml

#stash staged changes
git stash push --staged

#pass changes to branch created from main
git checkout --force -B main refs/remotes/origin/main
git checkout -B ${BUMP_KMC_BRANCH_NAME}

#apply stashed changes
git stash apply
git add resources/kyma-metrics-collector/values.yaml
git add resources/kyma-metrics-collector/Chart.yaml

#configure git
git config --global user.email ${GIT_EMAIL}
git config --global user.name ${GIT_NAME}

#commit and push changes
git commit -m "Bump KMC images and Chart to ${TAG}"
git remote set-url origin https://x-access-token:${GH_TOKEN}@github.com/${KYMA_METRICS_COLLECTOR_REPO}.git
git push --set-upstream origin ${BUMP_KMC_BRANCH_NAME} -f

#create PR
pr_link=$(gh pr create -B main --title "chore: bump KMC images and Chart to ${TAG}" --body "https://github.com/${KYMA_METRICS_COLLECTOR_REPO}/releases/tag/${TAG}" | tail -n 1)
echo "Link for created PR: ${pr_link}"

pr_number=$(echo "$pr_link" | awk -F'/' '{print $NF}')
gh pr edit $pr_number --add-label kind/enhancement
echo "$pr_number"
