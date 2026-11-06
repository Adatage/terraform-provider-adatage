# Releasing the provider

A release is a change of `VERSION` on `main`. The pipeline then does everything:

1. `ci/apps/terraform-provider-adatage.yml` runs `ci/mirror.sh`: the folder `apps/terraform-provider-adatage` (with its history) is pushed to `main` of the public GitHub repository, and when `v<VERSION>` does not exist there yet it is tagged.
2. The tag starts `.github/workflows/release.yml` on GitHub: tests, then GoReleaser builds the zips for every platform, signs `SHA256SUMS` with the GPG key and attaches the registry manifest to a GitHub release.
3. The Terraform Registry and the OpenTofu Registry notice the release and list the new version.

Use semantic versions. Without a bump of `VERSION` the code is mirrored but nothing is released.

## One-time setup

1. **GitHub repository**: a public repository named `terraform-provider-adatage` (the registries require this name) in the organization that is the provider's namespace, `adatage` by default. The address of the provider is `adatage/adatage`; if the organization or repository differs, change `TF_PROVIDER_GITHUB_REPO` in CI, the module path in `go.mod` and the imports, `Address` in `main.go` and `source` in the docs.
2. **GPG key** (both registries verify releases with it):
   ```
   gpg --full-generate-key        # RSA 4096, no passphrase-less keys
   gpg --armor --export-secret-keys <KEYID>   # -> GitHub secret GPG_PRIVATE_KEY
   gpg --armor --export <KEYID>               # -> the public key for the registries
   ```
   Add the repository secrets `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` on GitHub (Settings > Secrets and variables > Actions).
3. **GitLab CI variable** `GITHUB_TOKEN` (masked, protected): a GitHub token that can push code, workflow files and tags to the repository (a classic token with `repo` and `workflow`, or a fine-grained token with Contents and Workflows write access).
4. **Terraform Registry**: sign in at registry.terraform.io with GitHub, *Publish > Provider*, choose the repository, and add the **public** GPG key under the namespace's signing keys. The registry creates the webhook itself; later releases appear on their own.
5. **OpenTofu Registry**: submit the provider once through the registry's provider submission (an issue on github.com/opentofu/registry, which also asks for the public GPG key). Later releases are picked up from GitHub.

The first release can be made when the repository, the secrets and the variable exist: merge a commit with `VERSION` on `main`.

## Checking a release

`https://github.com/<repo>/releases/tag/v<VERSION>` lists the zips, `..._SHA256SUMS`, `..._SHA256SUMS.sig` and `..._manifest.json`. `terraform init` with `source = "adatage/adatage"` (or `tofu init`) installs it.
