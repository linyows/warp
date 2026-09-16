import Image from 'next/image'
import Link from 'next/link'
import type { Doc, Lang } from '@/lib/markdown'
import styles from '../page.module.css'

type Props = {
  doc: Doc
  lang: Lang
}

export const Site = ({ doc, lang }: Props) => {
  const y = new Date(Date.now()).getFullYear()

  return (
    <div className={styles.box}>
      <div className={styles.layout}>
        <header className={styles.header}>
          <div className={styles.logo}>
            <h1><Image src="/images/warp.svg" width={200} height={200} alt="Warp" priority /></h1>
          </div>
          <nav className={styles.langNav}>
            {lang === 'en' ? <span>English</span> : <Link href="/" hrefLang="en">English</Link>}
            {' | '}
            {lang === 'ja' ? <span>日本語</span> : <Link href="/ja" hrefLang="ja">日本語</Link>}
          </nav>
        </header>

        <div className={styles.page}>
          <div className={styles.content} lang={lang} dangerouslySetInnerHTML={{ __html: doc.html }} />
          <footer className={styles.footer}>
            <div className={styles.footerNav}>
              <a href="https://github.com/linyows/warp/issues" target="_blank" rel="noreferrer">Github Issues</a>
            </div>
            <div className={styles.copyright}>
              &copy; {y} <a href="https://github.com/linyows" target="_blank" rel="noreferrer">linyows</a>
            </div>
          </footer>
        </div>
      </div>
    </div>
  )
}
