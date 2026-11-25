# Language enforcement

`vcs` checks the words of each text before it sends the text. The check is the core of the tool.
It runs in the binary. It needs no Node, no Vale install, and no extra program on the path.

## What it does

Each verb that sends text runs the checkers first. Issue bodies, pull request bodies, comments,
and commit messages all count. A text with one finding goes nowhere. The verb exits 65 and prints
each finding on stderr.

| Checker | What it finds | Where the rules come from |
|---|---|---|
| lexi | Insensitive and inconsiderate words, and profanity | The rules of [alex](https://github.com/get-alex/alex), in a Go port |
| Vale | Prose that breaks the style rules, such as ASD-STE100 | The `.vale.ini` of the working directory |
| Credit guard | An AI credit line in a commit message or a pull request body | Built into `vcs` |

The checkers run in this order, and `vcs` sorts the findings by line and column. Each finding
names the checker, the position, the message, and the rule.

```text
vcs: lint: <checker>:<line>:<column>: <message> (<rule>)
vcs: lint: 1 finding(s). Nothing was sent
```

## Strict by design

- A checker that fails to run makes the verb exit 65. A text that no checker saw never goes out.
- A checker that can't start stops `vcs` with exit 69, before the verb runs.
- There is no warning tier. Vale reports each match, whatever its severity. Each match is a finding.
- An agent can't skip the check. The lint runs inside the verb, not in a separate script that an
  agent can turn off.

## How you set it up

| You want | Do this |
|---|---|
| Inclusive-language checks | Nothing. lexi always runs. Add an `.alexrc` in JSON or YAML to tune it |
| ASD-STE100 or another Vale style | Put a `.vale.ini` in the working directory. Vale runs when the file exists |
| No Vale | Leave out `.vale.ini`. Only lexi and the credit guard run |

lexi reads the same `.alexrc` options as alex. See [Dependencies](dependencies.md) for what each
project does, and [Configuration](configuration.md) for the variables.

## Why a controlled language

Agents and people read the text that `vcs` sends. A short, plain sentence leaves less room for a
wrong guess. The [ASD-STE100](dependencies.md#the-writing-standard) standard gives a clear rule set for such
text. The Vale styles turn those rules into checks. The checks run on every send, so the rules
hold without a person to remind the writer.
