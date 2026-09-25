import type { Preview } from '@storybook/react-vite'
import { ElisyumProvider } from '../src/providers'
import '../src/styles.css'

const preview: Preview = {
  globalTypes: {
    motion: {
      description: 'Motion preference',
      toolbar: {
        icon: 'time',
        items: ['system', 'full', 'reduced'],
      },
    },
    theme: {
      description: 'Color theme',
      toolbar: {
        icon: 'paintbrush',
        items: ['dark', 'light'],
      },
    },
  },
  initialGlobals: {
    motion: 'system',
    theme: 'dark',
  },
  decorators: [
    (Story, context) => (
      <ElisyumProvider motion={context.globals.motion} theme={context.globals.theme === 'light' ? 'light' : 'dark'}>
        <Story />
      </ElisyumProvider>
    ),
  ],
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
  },
}

export default preview
