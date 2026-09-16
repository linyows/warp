import { readFile } from 'node:fs/promises'
import path from 'node:path'
import type { Metadata } from 'next'
import matter from 'gray-matter'
import rehypeSlug from 'rehype-slug'
import rehypeStringify from 'rehype-stringify'
import remarkGfm from 'remark-gfm'
import remarkParse from 'remark-parse'
import remarkRehype from 'remark-rehype'
import { unified } from 'unified'

const contentDir = path.join(process.cwd(), 'content')

export type Lang = 'en' | 'ja'

export const paths: Record<Lang, string> = {
  en: '/',
  ja: '/ja',
}

export type Doc = {
  lang: Lang
  title: string
  description?: string
  html: string
}

const processor = unified()
  .use(remarkParse)
  .use(remarkGfm)
  .use(remarkRehype)
  .use(rehypeSlug)
  .use(rehypeStringify)

export async function loadDoc(lang: Lang): Promise<Doc> {
  const raw = await readFile(path.join(contentDir, `${lang}.md`), 'utf8')
  const { content, data } = matter(raw)
  const file = await processor.process(content)

  return {
    lang,
    title: typeof data.title === 'string' ? data.title : 'Warp',
    description: typeof data.description === 'string' ? data.description : undefined,
    html: String(file),
  }
}

export function metadataFor(doc: Doc): Metadata {
  return {
    title: doc.title,
    description: doc.description,
    alternates: {
      canonical: paths[doc.lang],
      languages: paths,
    },
  }
}
