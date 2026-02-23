import Image from 'next/image'
import { FetchBlocks, FetchPage } from 'rotion'
import { Page } from './components/Page'
import styles from './page.module.css'

export default async function Home() {
  const id = process.env.HOMEPAGE_ID as string
  const page = await FetchPage({ page_id: id, last_edited_time: 'force' })
  const logo = page.cover?.src || ''
  const icon = page.icon!.src
  const blocks = await FetchBlocks({ block_id: id, last_edited_time: page.last_edited_time })
  const y = new Date(Date.now()).getFullYear()

  return (
    <div className={styles.box}>
      <div className={styles.layout}>
        <header className={styles.header}>
          <div className={styles.logo}>
            <h1><Image src={logo} width={200} height={200} alt="Warp" /></h1>
          </div>
        </header>

        <div className={styles.page}>
          <Page blocks={blocks} />
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
