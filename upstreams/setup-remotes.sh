#!/bin/bash
# Set up all 4 remotes for the LLMOrchestrator repository
set -e

echo "Setting up remotes..."

# GitHub
git remote add github-vasic https://github.com/vasic-digital/LLMOrchestrator.git 2>/dev/null || echo "github-vasic already exists"
git remote add github-helix https://github.com/HelixDevelopment/LLMOrchestrator.git 2>/dev/null || echo "github-helix already exists"

# GitLab
git remote add gitlab-vasic https://gitlab.com/vasic-digital/LLMOrchestrator.git 2>/dev/null || echo "gitlab-vasic already exists"
git remote add gitlab-helix https://gitlab.com/HelixDevelopment/LLMOrchestrator.git 2>/dev/null || echo "gitlab-helix already exists"

echo "Remotes configured:"
git remote -v
