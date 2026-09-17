import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Dialog } from '../dialog/dialog'
import { Select } from './select'

describe('Select', () => {
  it('opens a long option list and keeps scrolling on the list', async () => {
    const options = Array.from({ length: 100 }, (_, index) => ({
      label: `Option ${index + 1}`,
      value: `option-${index + 1}`,
    }))
    options.unshift({ label: 'All categories', value: '' })

    render(<Select value="" options={options} />)

    fireEvent.mouseDown(screen.getByRole('combobox'))

    const list = await screen.findByRole('listbox')
    expect(list).toBeVisible()
    expect(list).toHaveStyle({
      maxHeight: 'min(var(--available-height), 20rem)',
      overflowY: 'auto',
      overscrollBehavior: 'contain',
    })
  })

  it('opens above the dialog layer', async () => {
    render(
      <Dialog open showHeader={false}>
        <Select options={[{ label: 'Database', value: 'database' }]} />
      </Dialog>,
    )

    fireEvent.mouseDown(screen.getByRole('combobox'))

    await screen.findByRole('listbox')
    const positioner = Array.from(
      document.querySelectorAll<HTMLElement>('*'),
    ).find((element) => element.className.toString().includes('z-[2147483002]'))
    expect(positioner).toBeTruthy()
  })
})
