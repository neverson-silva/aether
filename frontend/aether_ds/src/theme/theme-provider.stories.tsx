import type { Meta, StoryObj } from '@storybook/react'
import { Button } from '../components/button/button'
import { ThemeProvider, useTheme } from './theme-provider'

const ThemePreview = () => {
  const { theme, resolvedTheme, setTheme } = useTheme()
  return (
    <div className="space-y-4">
      <p className="text-body-md">
        Selected: {theme} / Resolved: {resolvedTheme}
      </p>
      <div className="flex gap-2">
        <Button size="sm" variant="outline" onClick={() => setTheme('light')}>
          Light
        </Button>
        <Button size="sm" variant="outline" onClick={() => setTheme('dark')}>
          Dark
        </Button>
        <Button size="sm" variant="outline" onClick={() => setTheme('system')}>
          System
        </Button>
      </div>
    </div>
  )
}

const meta = {
  title: 'Foundations/Theme Provider',
  component: ThemeProvider,
  tags: ['autodocs'],
  parameters: { layout: 'centered' },
} satisfies Meta<typeof ThemeProvider>
export default meta
type Story = StoryObj<typeof meta>
export const Dark: Story = {
  args: { defaultTheme: 'dark', persist: false, children: <ThemePreview /> },
}
export const Light: Story = {
  args: { defaultTheme: 'light', persist: false, children: <ThemePreview /> },
}
export const System: Story = {
  args: { defaultTheme: 'system', persist: false, children: <ThemePreview /> },
}
export const CustomizedTokens: Story = {
  args: {
    defaultTheme: 'light',
    persist: false,
    config: {
      semantics: {
        actionPrimary: '#173f9f',
        actionPrimaryHover: '#123681',
        actionPrimaryActive: '#0d2d6e',
      },
      typography: {
        familySans: '"IBM Plex Sans", sans-serif',
        sizeBodyMd: '15px',
      },
      radii: {
        control: '0.75rem',
        card: '1rem',
      },
      light: {
        semantics: {
          surfaceRaised: '#f3f6fb',
        },
      },
      dark: {
        semantics: {
          actionPrimary: '#4f83e8',
        },
      },
    },
    children: <ThemePreview />,
  },
}
