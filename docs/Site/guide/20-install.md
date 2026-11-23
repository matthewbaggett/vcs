# Install

Each release holds six binaries and one checksum file. Pick the one for your system.

## Platforms

| System | Chip | File |
|---|---|---|
| Linux | x86-64 | `vcs_<version>_linux_amd64` |
| Linux | Arm | `vcs_<version>_linux_arm64` |
| macOS | Intel | `vcs_<version>_darwin_amd64` |
| macOS | Apple silicon | `vcs_<version>_darwin_arm64` |
| Windows | x86-64 | `vcs_<version>_windows_amd64.exe` |
| Windows | Arm | `vcs_<version>_windows_arm64.exe` |

The checksum file is `vcs_<version>_checksums.txt`. It lists the sha256 sum of each binary.

## Steps

1. Open the releases page of `github.com/matthewbaggett/vcs`.
2. Download the binary for your system and the checksum file.
3. Check the sum. On Linux, run `sha256sum --check --ignore-missing vcs_<version>_checksums.txt`.
   On macOS, run `shasum -a 256 -c` on the same file.
4. Make the file run: `chmod +x vcs_<version>_<os>_<arch>`.
5. Move it onto your `PATH` and name it `vcs`.
6. Run `vcs session status` to see that it starts.

## Dev builds

A push to `main` makes a pre-release named `dev-<short sha>`. It holds the same six binaries. Use
it to try a change before the owner tags a release.

## Build it yourself

You need the latest stable Go. Run this in a checkout:

```bash
go build ./cmd/vcs
```

The release build also sets `CGO_ENABLED=0`, `-trimpath`, and the version string. See [the release
pipeline](release-pipeline.md).
