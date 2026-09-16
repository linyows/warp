import type { Metadata } from 'next'
import { Site } from '../components/Site'
import { loadDoc, metadataFor } from '@/lib/markdown'

export async function generateMetadata(): Promise<Metadata> {
  return metadataFor(await loadDoc('ja'))
}

export default async function Ja() {
  return <Site doc={await loadDoc('ja')} lang="ja" />
}
