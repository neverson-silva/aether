import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn } from 'storybook/test'
import { Wizard } from './wizard'

const meta = {
  component: Wizard,
  title: 'Patterns/Wizard',
  parameters: { layout: 'centered' },
} satisfies Meta<typeof Wizard>
export default meta
type Story = StoryObj<typeof meta>

const serviceSteps = [
  {
    id: 'source',
    label: 'Source',
    description: 'Connect the repository that contains your service.',
    summary: 'Repository and branch',
    content: (
      <div className="grid gap-5">
        <div>
          <p className="text-label text-text-primary">Repository</p>
          <p className="mt-1 text-body text-text-secondary">
            github.com/acme/payments-api
          </p>
        </div>
        <div>
          <p className="text-label text-text-primary">Branch</p>
          <p className="mt-1 text-body text-text-secondary">main</p>
        </div>
        <div className="rounded-lg border border-border-subtle bg-surface-2 p-4 text-supporting text-text-secondary">
          The service will deploy when changes land on the selected branch.
        </div>
      </div>
    ),
  },
  {
    id: 'runtime',
    label: 'Runtime',
    description: 'Choose how the service should run in production.',
    summary: 'Node.js · 2 vCPU',
    content: (
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-action bg-action-soft p-4">
          <p className="text-label text-text-primary">Node.js</p>
          <p className="mt-1 text-supporting text-text-secondary">
            For JavaScript and TypeScript services.
          </p>
          <p className="mt-4 font-technical text-log text-action-strong">
            v22 · 2 vCPU · 2 GB
          </p>
        </div>
        <div className="rounded-lg border border-border-subtle bg-surface-2 p-4">
          <p className="text-label text-text-primary">Container</p>
          <p className="mt-1 text-supporting text-text-secondary">
            Bring your own Dockerfile and runtime.
          </p>
        </div>
      </div>
    ),
  },
  {
    id: 'settings',
    label: 'Settings',
    description: 'Give the service a name and configure its public access.',
    summary: 'payments-api · public',
    content: (
      <div className="grid gap-5">
        <div>
          <p className="text-label text-text-primary">Service name</p>
          <div className="mt-2 min-h-10 rounded-md border border-border-default bg-field px-3 py-2 text-body text-text-primary">
            payments-api
          </div>
        </div>
        <div>
          <p className="text-label text-text-primary">Public URL</p>
          <div className="mt-2 min-h-10 rounded-md border border-border-default bg-field px-3 py-2 text-body text-text-primary">
            payments-api.aether.app
          </div>
        </div>
      </div>
    ),
  },
  {
    id: 'review',
    label: 'Review',
    description: 'Confirm the configuration before creating the service.',
    summary: 'Ready to create',
    content: (
      <div className="grid gap-3">
        <div className="flex items-center justify-between border-b border-border-subtle pb-3">
          <span className="text-supporting text-text-tertiary">Service</span>
          <span className="text-body text-text-primary">payments-api</span>
        </div>
        <div className="flex items-center justify-between border-b border-border-subtle pb-3">
          <span className="text-supporting text-text-tertiary">Runtime</span>
          <span className="text-body text-text-primary">Node.js 22</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-supporting text-text-tertiary">Access</span>
          <span className="text-body text-text-primary">Public</span>
        </div>
      </div>
    ),
  },
]

export const CreateService: Story = {
  args: { steps: serviceSteps },
  render: () => {
    const [activeStep, setActiveStep] = useState(0)
    return (
      <Wizard
        completeLabel="Create service"
        onComplete={fn()}
        onStepChange={setActiveStep}
        steps={serviceSteps}
        activeStep={activeStep}
      />
    )
  },
  play: async ({ canvas }) => {
    await canvas.getByRole('button', { name: 'Continue' }).click()
    await expect(canvas.getByRole('heading', { name: 'Runtime' })).toBeVisible()
  },
}
export const NeedsAttention: Story = {
  args: {
    activeStep: 1,
    completeLabel: 'Create project',
    steps: [
      {
        id: 'details',
        label: 'Project details',
        description: 'Name and describe the project.',
        summary: 'Missing project name',
        status: 'error',
        content: (
          <div className="rounded-lg border border-danger/60 bg-danger/10 p-4 text-body text-danger-strong">
            Add a project name before continuing.
          </div>
        ),
      },
      {
        id: 'members',
        label: 'Members',
        description: 'Invite people who need access.',
        summary: '3 members',
        content: (
          <p className="text-body text-text-secondary">Choose project members.</p>
        ),
      },
      {
        id: 'review',
        label: 'Review',
        description: 'Confirm the project configuration.',
        summary: 'Not ready',
        content: (
          <p className="text-body text-text-secondary">
            Review the project before creating it.
          </p>
        ),
      },
    ],
  },
}
