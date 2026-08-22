import { fireEvent, render, screen } from '@testing-library/react'
import { AlertDialog, AetherProvider, AppHeader, Button, Dialog, IconButton } from '@aether/design-system'
import { MagnifyingGlass } from '@phosphor-icons/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useState } from 'react'
import { CommandPalette, usePalette } from './command-palette'
import { CreateServiceLauncher } from './CreateServiceLauncher'

vi.mock('../hooks', () => ({
  useProjects: () => ({ data: [] }),
}))

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}))

vi.mock('../api/client', () => ({
  getServer: () => 'http://localhost:8080',
}))

describe('web command palette integration', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('opens the global command palette through the DS provider', async () => {
    function OpenButton() {
      const { setOpen } = usePalette()
      return <button type="button" onClick={() => setOpen(true)}>Open search</button>
    }

    function TestHarness() {
      return (
        <AetherProvider defaultTheme="dark" persist={false}>
          <OpenButton />
          <CommandPalette />
        </AetherProvider>
      )
    }

    render(<TestHarness />)
    fireEvent.click(screen.getByRole('button', { name: 'Open search' }))

    expect(await screen.findByRole('textbox', { name: 'Search or create...' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'ProjectsGo to' })).toBeVisible()
  })

  it('renders the create service palette when opened', async () => {
    render(
      <AetherProvider defaultTheme="dark" persist={false}>
        <CreateServiceLauncher open onClose={() => undefined} />
      </AetherProvider>,
    )

    expect(await screen.findByRole('textbox', { name: 'Search or create...' })).toBeVisible()
    expect(screen.getByText('Web application')).toBeVisible()
  })

  it('opens from the DS icon button used by the shell header', async () => {
    function HeaderSearch() {
      const { setOpen } = usePalette()
      return <IconButton label="Search" icon={MagnifyingGlass} onClick={() => setOpen(true)} />
    }

    render(
      <AetherProvider defaultTheme="dark" persist={false}>
        <HeaderSearch />
        <CommandPalette />
      </AetherProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(await screen.findByRole('textbox', { name: 'Search or create...' })).toBeVisible()
  })

  it('opens from the complete DS app header composition', async () => {
    function HeaderHarness() {
      const { setOpen } = usePalette()
      return (
        <>
          <AppHeader
            workspace="My Organization"
            search={<IconButton label="Search" icon={MagnifyingGlass} onClick={() => setOpen(true)} />}
          />
          <CommandPalette />
        </>
      )
    }

    render(
      <AetherProvider defaultTheme="dark" persist={false}>
        <HeaderHarness />
      </AetherProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Search' }))

    expect(await screen.findByRole('textbox', { name: 'Search or create...' })).toBeVisible()
  })

  it('opens the DS dialogs through the web bundle', async () => {
    function DialogHarness() {
      const [open, setOpen] = useState(false)
      const [alertOpen, setAlertOpen] = useState(false)
      return (
        <AetherProvider defaultTheme="dark" persist={false}>
          <Dialog open={open} onOpenChange={setOpen} title="Web dialog" trigger={<Button>Open web dialog</Button>}>
            <p>Web dialog content</p>
          </Dialog>
          <AlertDialog open={alertOpen} onOpenChange={setAlertOpen} trigger={<Button>Open web alert</Button>} title="Web alert" description="Web alert content" />
        </AetherProvider>
      )
    }

    render(<DialogHarness />)
    fireEvent.click(screen.getByRole('button', { name: 'Open web dialog' }))
    expect(await screen.findByText('Web dialog content')).toBeVisible()
    fireEvent.keyDown(document, { key: 'Escape' })
    fireEvent.click(screen.getByRole('button', { name: 'Open web alert' }))
    expect(await screen.findByText('Web alert content')).toBeVisible()
  })
})
