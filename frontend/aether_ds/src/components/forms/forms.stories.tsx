import type { Meta, StoryObj } from '@storybook/react'
import { Calendar } from '../calendar/calendar'
import { Checkbox } from '../checkbox/checkbox'
import { Combobox } from '../combobox/combobox'
import { DatePicker } from '../date-picker/date-picker'
import { DateRangePicker } from '../date-range-picker/date-range-picker'
import { FileUpload } from '../file-upload/file-upload'
import { FormActions } from '../form-actions/form-actions'
import { Input } from '../input/input'
import { InputGroup } from '../input-group/input-group'
import { InputOTP } from '../input-otp/input-otp'
import { NumberField } from '../number-field/number-field'
import { RadioGroup } from '../radio-group/radio-group'
import { Select } from '../select/select'
import { SelectSearch } from '../select-search/select-search'
import { Slider } from '../slider/slider'
import { Switch } from '../switch/switch'
import { Textarea } from '../textarea/textarea'
import { TimePicker } from '../time-picker/time-picker'
import { Toggle } from '../toggle/toggle'
import { ToggleGroup } from '../toggle-group/toggle-group'

const options = [
  { value: 'production', label: 'Production' },
  { value: 'staging', label: 'Staging' },
  { value: 'development', label: 'Development', disabled: true },
]
const meta = {
  title: 'Forms/Overview',
  component: Input,
  tags: ['autodocs'],
} satisfies Meta<typeof Input>
export default meta
type Story = StoryObj<typeof meta>

export const InputStates: Story = {
  render: () => (
    <div className="max-w-xl space-y-4">
      <Input
        label="Service name"
        placeholder="aether-api"
        description="Use a stable service identifier."
      />
      <Input
        label="Search"
        placeholder="Find a service"
        error="Service name is required."
      />
      <Input label="Loading" loading placeholder="Checking availability" />
      <Input
        label="Clearable"
        clearable
        onClear={() => undefined}
        defaultValue="production"
      />
    </div>
  ),
}
export const TextareaStates: Story = {
  render: () => (
    <div className="max-w-xl space-y-4">
      <Textarea
        label="Description"
        placeholder="Describe the deployment"
        showCount
        maxLength={160}
      />
      <Textarea
        label="Configuration"
        code
        error="Invalid YAML"
        defaultValue="service: aether-api"
      />
    </div>
  ),
}
export const GroupAndNumber: Story = {
  render: () => (
    <div className="max-w-xl space-y-4">
      <InputGroup prefix="https://" suffix=".aether.dev">
        <Input aria-label="Service hostname" placeholder="service" />
      </InputGroup>
      <NumberField label="Replicas" defaultValue={2} min={1} max={20} />
    </div>
  ),
}
export const ChoiceFields: Story = {
  render: () => (
    <div className="space-y-5">
      <Checkbox
        label="Enable production protection"
        description="Requires approval before deployment."
      />
      <RadioGroup
        label="Environment"
        options={options}
        defaultValue="production"
      />
      <Switch
        label="Automatic deploys"
        description="Deploy every successful main branch build."
        defaultChecked
      />
      <Toggle>Preview logs</Toggle>
      <ToggleGroup
        options={[
          { value: 'all', label: 'All' },
          { value: 'errors', label: 'Errors' },
          { value: 'warnings', label: 'Warnings' },
        ]}
      />
    </div>
  ),
}
export const SliderStates: Story = {
  render: () => (
    <div className="max-w-xl space-y-4">
      <Slider
        label="CPU allocation"
        min={0}
        max={100}
        step={5}
        defaultValue={40}
        marks={[25, 50, 75]}
      />
      <Slider
        label="Invalid allocation"
        error="Value exceeds environment limit"
        min={0}
        max={100}
        defaultValue={95}
      />
    </div>
  ),
}
export const SelectionFields: Story = {
  render: () => (
    <div className="max-w-xl space-y-4">
      <Select
        label="Environment"
        placeholder="Select an environment"
        options={options}
      />
      <SelectSearch
        label="Search environment"
        placeholder="Type to search"
        options={options}
      />
      <Combobox
        label="Service"
        placeholder="Search services"
        options={[
          { value: 'api', label: 'Aether API' },
          { value: 'web', label: 'Aether Web' },
          { value: 'worker', label: 'Aether Worker' },
        ]}
      />
    </div>
  ),
}
export const DateAndTime: Story = {
  render: () => (
    <div className="max-w-xl space-y-4">
      <DatePicker label="Release date" />
      <DateRangePicker
        label="Maintenance window"
        presets={[
          { label: 'This week', start: '2026-08-17', end: '2026-08-23' },
        ]}
      />
      <TimePicker label="Start time" timezone="America/Sao_Paulo" withSeconds />
      <Calendar />
      <InputOTP label="Verification code" error="Code expired" />
      <FormActions dirty success="Draft saved" />
    </div>
  ),
}
export const Upload: Story = {
  render: () => (
    <FileUpload
      accept=".yaml,.json"
      multiple
      maxSize={5_000_000}
      onFilesChange={() => undefined}
      onError={() => undefined}
    />
  ),
}
