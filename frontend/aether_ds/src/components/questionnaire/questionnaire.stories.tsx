import type { Meta, StoryObj } from '@storybook/react'
import { Questionnaire } from './questionnaire'

const meta = {
  title: 'Patterns/Questionnaire',
  component: Questionnaire,
  tags: ['autodocs'],
} satisfies Meta<typeof Questionnaire>
export default meta
type Story = StoryObj<typeof meta>
export const Onboarding: Story = {
  args: {
    questions: [
      {
        id: 'scale',
        title: 'How large is your service?',
        options: [
          { value: 'small', label: 'Small' },
          { value: 'large', label: 'Large' },
        ],
      },
      {
        id: 'region',
        title: 'Preferred region',
        options: [
          { value: 'sa', label: 'São Paulo' },
          { value: 'us', label: 'Virginia' },
        ],
        when: (answers) => answers.scale === 'large',
      },
    ],
  },
}
