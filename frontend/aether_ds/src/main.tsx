import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Button, ThemeProvider } from './index'

function App() {
  return (
    <ThemeProvider>
      <main className="min-h-screen bg-background px-6 py-16 text-foreground">
        <div className="mx-auto max-w-3xl space-y-8">
          <p className="font-label-caps text-primary">AETHER / DESIGN SYSTEM</p>
          <h1 className="font-display-lg">
            Infrastructure, with a point of view.
          </h1>
          <p className="max-w-xl text-body-md text-muted-foreground">
            A React-first component library for the Aether platform.
          </p>
          <div className="flex gap-3">
            <Button>Explore components</Button>
            <Button variant="secondary">View tokens</Button>
          </div>
        </div>
      </main>
    </ThemeProvider>
  )
}

const root = document.getElementById('root')

if (!root) throw new Error('Aether root element was not found')

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
