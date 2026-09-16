Docs
==

The source of <https://warp.linyo.ws>, a static site built with [Next.js](https://nextjs.org/).

Content
--

The pages are plain Markdown files under `content/`, managed in this repository.
Each file starts with front matter:

```markdown
---
title: Warp
description: Warp is an outbound transparent SMTP proxy.
---

**WARP** is an outbound **transparent** SMTP proxy.
```

One file per language: `content/en.md` is rendered at `/` and `content/ja.md` at
`/ja`. GitHub Flavored Markdown (tables, strikethrough, autolinks) is supported.
The two files are separate documents, not a translation pipeline — keep them in
step by hand when the content changes.

Adding a language means adding `content/<lang>.md`, extending `Lang` and `paths`
in `lib/markdown.ts`, and adding a route under `app/`.

Images are not stored here: `npm run assets` copies `../misc/warp.svg` and
`../misc/architecture.png` into `public/images/`, so Markdown refers to them as
`/images/architecture.png`. Add new shared images to `../misc` and to the `assets`
script in `package.json`.

Development
--

```bash
npm install
npm run dev     # http://localhost:3000
npm run build   # static export into out/
```

Deployment
--

`.github/workflows/pages.yml` builds this directory and publishes `out/` to GitHub
Pages after a successful build on `main`. No API token or external service is
required.
