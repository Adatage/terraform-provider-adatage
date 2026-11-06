#!/bin/sh
# Publishes the provider: copies apps/terraform-provider-adatage (with its history) to the public GitHub repository and, when VERSION names
# a version that has no tag there yet, tags it. The tag starts .github/workflows/release.yml in that repository, which runs GoReleaser
# (signed zips and checksums); the Terraform and OpenTofu registries take the release from GitHub. See RELEASING.md.
#
# Needs the CI variable GITHUB_TOKEN (a token that may push to the repository, including workflow files and tags), and optionally
# TF_PROVIDER_GITHUB_REPO (default adatage/terraform-provider-adatage).
set -eu

: "${GITHUB_TOKEN:?set the CI variable GITHUB_TOKEN to a GitHub token that can push to the provider repository}"
repo="${TF_PROVIDER_GITHUB_REPO:-adatage/terraform-provider-adatage}"
dir=apps/terraform-provider-adatage
version="$(tr -d ' \n' <"$dir/VERSION")"
case "$version" in
[0-9]*.[0-9]*.[0-9]*) ;;
*)
	echo "VERSION must look like 1.2.3, got '$version'" >&2
	exit 1
	;;
esac

if [ "$(git rev-parse --is-shallow-repository)" = true ]; then
	git fetch --unshallow --quiet
fi
git subtree split --prefix="$dir" -b tf-provider-mirror >/dev/null
sha="$(git rev-parse tf-provider-mirror)"

url="https://x-access-token:${GITHUB_TOKEN}@github.com/${repo}.git"
git push --quiet "$url" "tf-provider-mirror:refs/heads/main"
echo "mirrored ${dir} to github.com/${repo} at ${sha}"

if git ls-remote --exit-code --tags "$url" "refs/tags/v${version}" >/dev/null 2>&1; then
	echo "v${version} exists already: bump apps/terraform-provider-adatage/VERSION for a new release"
else
	git push --quiet "$url" "tf-provider-mirror:refs/tags/v${version}"
	echo "tagged v${version}: the release workflow on GitHub publishes it"
fi
