import type { Metadata } from 'next'
import { Site } from './components/Site'
import { loadDoc, metadataFor } from '@/lib/markdown'

export async function generateMetadata(): Promise<Metadata> {
  return metadataFor(await loadDoc('en'))
}

export default async function Home() {
  return <Site doc={await loadDoc('en')} lang="en" />
}
