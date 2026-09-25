import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ElisyumPreview } from './app'
import { ElisyumProvider } from './providers'
import './styles.css'

const root = document.getElementById('root')

if (!root) {
  throw new Error('Elisyum root element was not found')
}

createRoot(root).render(
  <StrictMode>
    <ElisyumProvider>
      <ElisyumPreview />
    </ElisyumProvider>
  </StrictMode>,
)
