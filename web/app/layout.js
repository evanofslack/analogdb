import { ClientProviders } from './providers/client-providers'
import { NuqsAdapter } from 'nuqs/adapters/next/app'
import '../styles/globals.css'

export const metadata = {
    title: 'AnalogDB',
    description: 'Film photography community archive',
}

export default function RootLayout({ children }) {
    return (
        <html lang="en">
            <body>
                <NuqsAdapter>
                    <ClientProviders>
                        {children}
                    </ClientProviders>
                </NuqsAdapter>
            </body>
        </html>
    )
}
