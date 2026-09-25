import { ElisyumProvider } from '@aether/elisyum-ds'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createRouter, RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { OrgProvider } from './components/OrgProvider'
import { RealtimeProvider } from './components/RealtimeProvider'
import { routeTree } from './routeTree.gen'
import '@aether/elisyum-ds/styles.css'
import './styles.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 2000, refetchOnWindowFocus: false },
  },
})

const router = createRouter({
  routeTree,
  defaultPreload: false,
  context: { queryClient },
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

const root = document.getElementById('root')

if (!root) {
  throw new Error('Aether root was not found')
}

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ElisyumProvider
        defaultTheme="dark"
        density="standard"
      >
        <RealtimeProvider>
          <OrgProvider>
            <RouterProvider router={router} />
          </OrgProvider>
        </RealtimeProvider>
      </ElisyumProvider>
    </QueryClientProvider>
  </StrictMode>,
)
