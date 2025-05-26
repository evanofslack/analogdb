import { ClientProviders } from './providers/client-providers'
import '../styles/globals.css'

export const metadata = {
  title: 'AnalogDB',
  description: 'Film photography community archive',
}

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>
        <ClientProviders>
          {children}
        </ClientProviders>
      </body>
    </html>
  )
}
