# Dependencies

`vcs` builds on four projects. Three are Go modules in `go.mod`. One is a standard that the rules
enforce. Each section says what the project does in `vcs`, and where `vcs` uses it.

| Project | Kind | Used for | Where in `vcs` |
|---|---|---|---|
| [lexi](https://github.com/matthewbaggett/lexi) | Go module | Inclusive-language check | `pkg/lint`, on every send |
| [Vale](https://github.com/matthewbaggett/vale) fork | Go module | Style check | `pkg/lint`, when `.vale.ini` exists |
| [goldmark](https://github.com/yuin/goldmark) | Go module | Markdown to HTML | `pkg/site`, in `vcs site gen` |
| [ASD-STE100](https://www.asd-ste100.org/) | Standard | The writing rules | The Vale styles of the project |

## The lexi checker

lexi is a static Go port of [alex](https://github.com/get-alex/alex), the inclusive-language
linter. `vcs` calls lexi in process, in place of Node alex. The binary needs no Node.

- **What it finds.** Words that hurt, exclude, or insult. It also finds profanity. The
  `profanitySureness` setting of `.alexrc` sets how sure a match must be.
- **How `vcs` uses it.** On start, `vcs` reads the `.alexrc` of the working directory. It builds
  one lexi checker. It scans each text as Markdown.
- **Parity.** lexi matches alex 11.0.1. It uses `retext-equality` 6.6.0 and `retext-profanities`
  7.2.2. A corpus of more than 30,000 recorded cases checks the match.
- **Gaps.** lexi reads `.alexrc` as JSON or YAML only. It doesn't run `.alexrc.js`.

Source: [github.com/matthewbaggett/lexi](https://github.com/matthewbaggett/lexi). Upstream:
[github.com/get-alex/alex](https://github.com/get-alex/alex).

## Vale

[Vale](https://github.com/vale-cli/vale) is a linter for prose. It reads a set of style rules and
reports each place where a text breaks one. `vcs` uses a fork, `github.com/matthewbaggett/vale/v3`,
at version `v3.28.2-fork.1`.

- **What the fork adds.** A Go package, `pkg/vale`, that lints a string in process. Upstream Vale
  is a command. The fork lets `vcs` call it as a library.
- **How `vcs` uses it.** When the working directory holds a `.vale.ini`, `vcs` loads that file. It
  lints each text as Markdown. Each match becomes a finding, whatever its severity.
- **What the rules are.** The `.vale.ini` of the project names them. The ASD-STE100 styles are
  one set.

Source: [github.com/matthewbaggett/vale](https://github.com/matthewbaggett/vale). Upstream:
[github.com/vale-cli/vale](https://github.com/vale-cli/vale).

## The goldmark renderer

[goldmark](https://github.com/yuin/goldmark) is a Markdown parser and renderer for Go. It follows
the CommonMark rules.

- **How `vcs` uses it.** `vcs site gen` turns the Markdown pages of `docs/Site` into the HTML pages
  you read now. It turns on the table extension and automatic heading IDs.
- **What it doesn't do.** goldmark has no part in the language check. It builds this site only.

Source: [github.com/yuin/goldmark](https://github.com/yuin/goldmark).

## The writing standard

ASD-STE100 is a standard for controlled technical English. The Aerospace, Security and Defence
Industries Association of Europe (ASD) publishes it. Aircraft maintenance manuals use it. Its aim
is text that a reader with limited English can follow without doubt.

- **What the rules ask for.** Short sentences. One fact or one step in each sentence. Active
  voice. Plain, consistent words.
- **How `vcs` enforces it.** The Vale styles of the project turn the rules into checks. `vcs` runs
  those checks on each text that it sends.
- **A limit.** The styles are a practical adaptation. The standard itself is larger. The
  official edition wins when the two disagree.

Source: [asd-ste100.org](https://www.asd-ste100.org/).

## Where to go next

Read [Language enforcement](language-enforcement.md) for the full flow of a check.
