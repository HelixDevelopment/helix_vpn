#!/bin/bash
# Sync (fetch) from all remotes
set -e

echo "Fetching from all remotes..."

for remote in $(git remote); do
    echo "  <- $remote"
    git fetch "$remote" 2>&1 || echo "    Warning: fetch from $remote failed"
done

echo "Done."
