# Install

Each release holds six binaries and one checksum file. Pick the one for your system.

## Download

The latest release lives on the public mirror at `github.com/matthewbaggett/vcs`. The links below
fill in the newest version when your browser can run scripts. Without a script, use [the latest
release](https://github.com/matthewbaggett/vcs/releases/latest).

<div id="downloads">
<p><a href="https://github.com/matthewbaggett/vcs/releases/latest">Latest release</a> and <a href="https://github.com/matthewbaggett/vcs/releases">all releases</a></p>
</div>
<script>
(function () {
  var box = document.getElementById("downloads");
  var names = [["Linux", "x86-64", "linux_amd64"], ["Linux", "Arm", "linux_arm64"], ["macOS", "Intel", "darwin_amd64"], ["macOS", "Apple silicon", "darwin_arm64"], ["Windows", "x86-64", "windows_amd64.exe"], ["Windows", "Arm", "windows_arm64.exe"]];
  function link(asset) { return '<a href="' + asset.browser_download_url + '">' + asset.name + "</a>"; }
  fetch("https://api.github.com/repos/matthewbaggett/vcs/releases?per_page=20")
    .then(function (r) { if (!r.ok) { throw new Error(r.status); } return r.json(); })
    .then(function (list) {
      var latest = list.filter(function (r) { return !r.prerelease && !r.draft; })[0];
      var dev = list.filter(function (r) { return r.prerelease && !r.draft; })[0];
      if (!latest) { return; }
      var rows = "";
      names.forEach(function (n) {
        var a = latest.assets.filter(function (x) { return x.name.slice(-n[2].length) === n[2]; })[0];
        rows += "<tr><td>" + n[0] + "</td><td>" + n[1] + "</td><td>" + (a ? link(a) : "none") + "</td></tr>";
      });
      var sums = latest.assets.filter(function (x) { return /checksums\.txt$/.test(x.name); })[0];
      var html = "<p>Latest release: <a href=\"" + latest.html_url + "\">" + latest.tag_name + "</a></p>";
      html += "<table><thead><tr><th>System</th><th>Chip</th><th>File</th></tr></thead><tbody>" + rows + "</tbody></table>";
      html += "<p>Checksums: " + (sums ? link(sums) : "none") + "</p>";
      if (dev) { html += "<p>Dev build: <a href=\"" + dev.html_url + "\">" + dev.tag_name + "</a></p>"; }
      box.innerHTML = html;
    })
    .catch(function () {});
})();
</script>

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
it to try a change before the owner tags a release. The download links of this page show the newest
pre-release when the mirror holds one.

## Build it yourself

You need the latest stable Go. Run this in a checkout:

```bash
go build ./cmd/vcs
```

The release build also sets `CGO_ENABLED=0`, `-trimpath`, and the version string. See [the release
pipeline](release-pipeline.md).
