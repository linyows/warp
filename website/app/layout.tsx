import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import 'rotion/style.css'
import './globals.css'

const inter = Inter({
  weight: ['400', '700'],
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-inter',
})

export const metadata: Metadata = {
  title: 'Warp',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en">
      <body className={inter.variable}>
        {children}
      </body>
    </html>
  )
}
