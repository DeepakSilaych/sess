# Releasing sess

Releases are built and published manually. There are no CI or release workflows.

## Version sources

- `internal/api/api.go`: `Version` is the client/helper release version.
- The same file's `Protocol` is the remote command protocol version; change it only when compatibility requires it.
- `ZMXVersion` pins the supported backend. Its release checksums and installation-version check live in `internal/provision/provision.go`.

Before tagging, update version references in the README, user guide, landing page, troubleshooting guide, and changelog. Do not label a backend version supported without testing its command/output contract.

## Validate and build

Start from a clean checkout of the intended release commit. Run local checks:

```sh
make test
make integration
./scripts/release.sh
```

The release script builds all four remote helpers, embeds them into four client targets, and packages documentation alongside each executable:

```text
sess-darwin-arm64.tar.gz
sess-darwin-amd64.tar.gz
sess-linux-arm64.tar.gz
sess-linux-amd64.tar.gz
SHA256SUMS
```

Verify the archive checksums from `dist/`:

```sh
cd dist
shasum -a 256 -c SHA256SUMS
```

Extract and smoke-test the executable for your own platform with `sess version` and `sess --help`. Check that the version matches the intended tag. Cross-compilation is not runtime testing: record which client/VM combinations were exercised.

## Tag and publish

For a release whose code reports `0.7.0`, the tag is `v0.7.0`. Use the actual version for a future release, and check that it is not already published.

```sh
git tag -a v0.7.0 -m "sess v0.7.0 — persistent SSH terminals powered by zmx"
git push origin main
git push origin v0.7.0

gh release create v0.7.0 --verify-tag --draft \
  --title "sess v0.7.0" --notes-file /path/to/release-notes.md \
  dist/sess-*.tar.gz dist/SHA256SUMS
```

Inspect the draft's tag, notes, platform filenames, and uploaded checksums, then publish:

```sh
gh release edit v0.7.0 --draft=false --latest
```

Do not move an already-published tag to another commit. Publish a new patch version for corrections to a release.

## After publishing

Verify that all four download links work, the uploaded checksum file matches the archives, and the repository description, topics, and homepage reflect the current product. Keep the user-facing release notes clear about migration and platform coverage.

The existing GitHub Pages site serves `docs/` from `main`. Documentation changes there use the repository's Pages configuration; do not add a workflow to publish the site.

## Homebrew and npm

Keep `npm/releases.json` aligned with the GitHub release version and its four archive checksums. The npm wrapper version in `package.json` may increase independently when only the launcher changes; `npm/releases.json` selects the actual sess binary version.

After publishing the binary release:

1. Update `Formula/sess.rb` in [homebrew-tap](https://github.com/DeepakSilaych/homebrew-tap) with the version, URLs, and all four checksums. Run `brew style deepaksilaych/tap/sess`, install it, and run `brew test deepaksilaych/tap/sess` before pushing.
2. Run `npm test`, then `npm pack --dry-run`. Test the packed archive with `npx --yes --package /absolute/path/to/sess-cli-VERSION.tgz sess version`.
3. Authenticate locally with `npm login`, then run `npm publish --access public`. Complete npm's authentication challenge if requested. No CI workflow is used.
4. Verify `npm view sess-cli version` and `npx --yes sess-cli@VERSION version` from a fresh cache.

The package provides the executable name `sess`; npm resolves the single bin entry when users run `npx sess-cli`. It has no install scripts or runtime dependencies. The launcher verifies the release archive before extracting the executable and passes the terminal directly to sess.
