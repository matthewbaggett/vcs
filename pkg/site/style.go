package site

// stylesheet is the one stylesheet of the site. It follows the colour scheme of the reader.
// On a narrow screen, the page list sits under the text, and a link in the header jumps to it.
const stylesheet = `:root {
  --bg: #ffffff;
  --fg: #1b1f24;
  --muted: #57606a;
  --line: #d0d7de;
  --accent: #0b5cad;
  --code: #f3f5f7;
  --head: #f6f8fa;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #14181d;
    --fg: #e6e9ed;
    --muted: #9aa4af;
    --line: #2f3740;
    --accent: #6cb2ff;
    --code: #1d242c;
    --head: #181e25;
  }
}
* { box-sizing: border-box; }
html { -webkit-text-size-adjust: 100%; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--fg);
  font: 16px/1.6 system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
}
a { color: var(--accent); }
.top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--head);
  border-bottom: 1px solid var(--line);
  position: sticky;
  top: 0;
  z-index: 1;
}
.brand { font-weight: 700; font-size: 1.1rem; color: var(--fg); text-decoration: none; }
.jump { font-size: 0.95rem; }
.layout { display: block; max-width: 1100px; margin: 0 auto; padding: 0 16px; }
main { min-width: 0; padding: 8px 0 32px; }
main h1 { font-size: 1.8rem; line-height: 1.25; margin: 1.2rem 0 0.8rem; overflow-wrap: anywhere; }
main h2 { font-size: 1.3rem; margin: 2rem 0 0.6rem; padding-bottom: 0.2rem; border-bottom: 1px solid var(--line); }
main h3 { font-size: 1.1rem; margin: 1.5rem 0 0.4rem; }
code, pre { font: 0.9rem/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
code { background: var(--code); padding: 0.1em 0.35em; border-radius: 4px; overflow-wrap: anywhere; }
pre { background: var(--code); padding: 12px 14px; border-radius: 6px; overflow-x: auto; border: 1px solid var(--line); }
pre code { background: none; padding: 0; overflow-wrap: normal; }
table { border-collapse: collapse; display: block; max-width: 100%; overflow-x: auto; margin: 1rem 0; }
th, td { border: 1px solid var(--line); padding: 6px 10px; text-align: left; vertical-align: top; }
th { background: var(--head); }
.side { border-top: 1px solid var(--line); padding: 16px 0 40px; font-size: 0.95rem; }
.side h2 { font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); margin: 1.2rem 0 0.3rem; }
.side ul { list-style: none; margin: 0; padding: 0; }
.side li { margin: 0; }
.side a { display: block; padding: 2px 0; text-decoration: none; }
.side a[aria-current="page"] { font-weight: 700; color: var(--fg); }
@media (min-width: 900px) {
  .jump { display: none; }
  .layout { display: grid; grid-template-columns: 250px minmax(0, 1fr); gap: 40px; }
  .side { order: -1; border-top: 0; border-right: 1px solid var(--line); padding-right: 16px;
          position: sticky; top: 52px; align-self: start; max-height: calc(100vh - 52px); overflow-y: auto; }
}
`
