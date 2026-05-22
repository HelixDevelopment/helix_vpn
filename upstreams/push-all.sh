#!/bin/bash
# Push to all 4 remotes (vasic-digital GitHub/GitLab + HelixDevelopment GitHub/GitLab)
set -e

echo "Pushing to all remotes..."

for remote in $(git remote); do
    echo "  -> $remote"
    git push "$remote" --all 2>&1 || echo "    Warning: push to $remote failed"
    git push "$remote" --tags 2>&1 || echo "    Warning: tags push to $remote failed"
done

echo "Done."
