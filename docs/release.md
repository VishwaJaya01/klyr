# Release Guide

This document describes how to cut a Klyr release.

## Prerequisites

- Clean working tree
- All tests and lint passing
- Updated `CHANGELOG.md`

## Steps

1) Bump version if needed (e.g., v0.1.0 already set):

```bash
git tag v0.1.0
```

2) Push commits and tags:

```bash
git push
git push --tags
```

3) Draft a GitHub release using the tag `v0.1.0` and paste the release notes from `CHANGELOG.md`.

## Release Notes

Use the `CHANGELOG.md` entry for the tagged version as the release notes.
