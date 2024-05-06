import type { GetStaticProps, InferGetStaticPropsType } from 'next'
import Image from 'next/image'
import Head from 'next/head'
import Link from 'next/link'
import {
  FetchBlocks,
  FetchPage,
  FetchBlocksRes,
} from 'rotion'
import { Page, Link as RotionLink } from 'rotion/ui'
import styles from '@/styles/Home.module.css'

type Props = {
  icon: string
  logo: string
  blocks: FetchBlocksRes
}

export const getStaticProps: GetStaticProps<Props> = async (context) => {
  const id = process.env.HOMEPAGE_ID as string
  const page = await FetchPage({ page_id: id, last_edited_time: 'force' })
  const logo = page.cover?.src || ''
  const icon = page.icon!.src
  const blocks = await FetchBlocks({ block_id: id, last_edited_time: page.last_edited_time })

  return {
    props: {
      icon,
      logo,
      blocks,
    }
  }
}

export default function Home({ logo, icon, blocks }: InferGetStaticPropsType<typeof getStaticProps>) {
  const y = new Date(Date.now()).getFullYear()
  return (
    <>
      <Head>
        <title>Warp</title>
        <link rel="icon" type="image/svg+xml" href={icon} />
      </Head>
      <div className={styles.box}>
        <div className={styles.layout}>
          <header className={styles.header}>
            <div className={styles.logo}>
              <h1><Image src={logo} width={200} height={200} alt="Warp" /></h1>
            </div>
          </header>

          <div className={styles.page}>
            <Page blocks={blocks} href="/[title]" link={Link as RotionLink} />
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
    </>
  )
}
