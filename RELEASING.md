# Releasing the provider

A release is a change of `VERSION` on `main`. The pipeline then does everything:

1. `ci/apps/terraform-provider-adatage.yml` runs `ci/mirror.sh`: the folder `apps/terraform-provider-adatage` (with its history) is pushed to `main` of the public GitHub repository, and when `v<VERSION>` does not exist there yet it is tagged.
2. The tag starts `.github/workflows/release.yml` on GitHub: tests, then GoReleaser builds the zips for every platform, signs `SHA256SUMS` with the GPG key and attaches the registry manifest to a GitHub release.
3. The Terraform Registry and the OpenTofu Registry notice the release and list the new version.

Use semantic versions. Without a bump of `VERSION` the code is mirrored but nothing is released.

## One-time setup

The repository is https://github.com/Adatage/terraform-provider-adatage (public, empty, no README/licence added by GitHub, so the first push is clean). Do the steps in this order.

### 1. The GPG key

Both registries check that a release was signed with a key you registered. Make one on your own machine (not on a server):

```
gpg --full-generate-key
```
Choose `(1) RSA and RSA`, size `4096`, expiry `0` (never) or `2y` if you accept renewing it, name "Adatage Cloud provider", the release email, and a **passphrase** (write it down; GitHub needs it). Then:

```
gpg --list-secret-keys --keyid-format=long      # the line "sec rsa4096/AAAAAAAAAAAAAAAA" gives the key id AAAAAAAAAAAAAAAA
gpg --armor --export-secret-keys AAAAAAAAAAAAAAAA > private.asc   # SECRET: for GitHub step 2, delete the file afterwards
gpg --armor --export AAAAAAAAAAAAAAAA > public.asc                # public: for both registries (steps 4 and 5)
```
Keep a backup of `private.asc` and the passphrase in your password manager; losing them means a new key and re-registering in both registries.

### 2. GitHub: secrets for the release workflow

In the repository: *Settings > Secrets and variables > Actions > New repository secret*:

| Name | Value |
|:-----|:------|
| `GPG_PRIVATE_KEY` | the whole content of `private.asc`, including the `-----BEGIN PGP PRIVATE KEY BLOCK-----` lines |
| `GPG_PASSPHRASE` | the passphrase of the key |

Also check *Settings > Actions > General > Workflow permissions*: "Read and write permissions" (the workflow asks for `contents: write` itself, but an organization policy can forbid it).

### 3. GitLab: the token the pipeline pushes with

1. On GitHub: *your avatar > Settings > Developer settings > Personal access tokens > Fine-grained tokens > Generate new token*. Owner: the **Adatage** organization (an organization owner may have to allow fine-grained tokens first: org *Settings > Personal access tokens*). Repository access: *Only select repositories* > `terraform-provider-adatage`. Permissions (Repository): **Contents: Read and write** and **Workflows: Read and write** (the mirror contains `.github/workflows`, and GitHub refuses a push that adds them without it). Expiry: 1 year; put the renewal date in the calendar. Copy the token (`github_pat_...`).
   If the organization does not allow fine-grained tokens, use a classic token (*Tokens (classic)*) with the scopes `repo` and `workflow`.
2. In GitLab: the project *Settings > CI/CD > Variables > Add variable*: key `GITHUB_TOKEN`, value the token, type *Variable*, **Mask variable** on, **Protect variable** on (the pipeline that publishes runs on `main`, which is protected), *Expand variable reference* off. Optional second variable `TF_PROVIDER_GITHUB_REPO` if the repository ever moves (default `Adatage/terraform-provider-adatage`).

With `glab` this is: `glab variable set GITHUB_TOKEN --masked --protected` (it asks for the value).

### 4. Terraform Registry

1. Go to https://registry.terraform.io and *Sign in* with GitHub. Authorize the HashiCorp app and **grant access to the Adatage organization** (GitHub: *Settings > Applications > Authorized OAuth Apps > Terraform Registry > Organization access > Grant*), otherwise the organization is not offered.
2. *Publish > Provider*, choose the Adatage organization and the repository `terraform-provider-adatage`, accept the terms.
3. First time only the registry asks for the signing key: *User Settings > Signing Keys* (https://registry.terraform.io/settings/gpg-keys), pick the namespace (Adatage), *New GPG key*, paste the content of `public.asc`.
4. The registry installs a webhook on the repository. It only lists versions of tags that have a release with the signed files, so the provider appears after the first release (see below). The address is then `source = "adatage/adatage"`; if the registry shows the namespace with a capital letter, use that spelling in `source` and in the docs.

### 5. OpenTofu Registry

OpenTofu has no web login: you submit two **GitHub issue forms** on github.com/opentofu/registry, in the browser. They must be filled in through the form UI; issues made with `gh`, the API or by hand are ignored and closed.

1. Make your membership of the Adatage organization public (GitHub: the organization's *People* page > your row > visibility *Public*). Validation fails otherwise.
2. **Signing key**: https://github.com/opentofu/registry/issues/new?template=provider_key.yml
   - Provider Namespace: `Adatage`
   - Provider Name: `adatage` (or leave empty to use the key for every provider of the organization)
   - tick "I have made my membership public"
   - Provider GPG Key: the whole content of `public.asc`
3. **Provider**: https://github.com/opentofu/registry/issues/new?template=provider.yml with the repository `Adatage/terraform-provider-adatage`.
4. The OpenTofu team reviews and approves; after that new GitHub releases are picked up on their own. Do both after the first release exists. The rules are in the registry's `POLICY.md`.

### 6. The first release

When steps 1 to 3 are done, push to `main` (or retry the failed `publish-terraform-provider` job). `VERSION` is `0.1.0`, so the pipeline pushes the code and the tag `v0.1.0`; the GitHub Actions run *release* takes a few minutes. Check *Actions* in the repository and then the *Releases* page. Do steps 4 and 5 once the release exists (the registries need a release to verify the repository and the key).

## Checking a release

`https://github.com/<repo>/releases/tag/v<VERSION>` lists the zips, `..._SHA256SUMS`, `..._SHA256SUMS.sig` and `..._manifest.json`. `terraform init` with `source = "adatage/adatage"` (or `tofu init`) installs it.
