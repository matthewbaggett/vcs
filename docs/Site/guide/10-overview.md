# Overview

`vcs` is one Go command for Forgejo, Gitea, and GitHub. It reads and writes pull requests, issues,
labels, milestones, boards, time, and CI runs. It also wraps the git steps of a normal change.

The command has no server step and no runtime. It's one static binary.

## Language enforcement

`vcs` checks the words of each text before it sends the text. The [lexi](dependencies.md#lexi)
checker finds insensitive words and profanity. It's a Go port of alex. [Vale](dependencies.md#vale)
applies the style rules, such as [ASD-STE100](dependencies.md#asd-ste100). A text with one finding goes nowhere, and
the verb exits 65. Both linters run inside the binary. See [Language
enforcement](language-enforcement.md).

## Why it exists

A team that works with agents needs one stable command. The command checks each text it sends. It
gives a short hint after each run, so an agent knows what to do next. It exits with the same codes
on every forge.

## Where to start

| You want to | Read |
|---|---|
| See how `vcs` checks language | [Language enforcement](language-enforcement.md) |
| See what `vcs` builds on | [Dependencies](dependencies.md) |
| Install the binary | [Install](install.md) |
| Run your first commands | [Quickstart](quickstart.md) |
| Learn how the drivers work | [Concepts](concepts.md) |
| Add a token or a variable | [Configuration](configuration.md) |
| Find one verb | [Commands](commands.md) |
| Read the exit codes | [Exit codes and hints](exit-codes.md) |

## Download

Get the binary for your system. See [Install](install.md) for the steps.

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

## Where the code lives

The code is public at `github.com/matthewbaggett/vcs`. That repository is a read-only mirror of the
canonical one.
