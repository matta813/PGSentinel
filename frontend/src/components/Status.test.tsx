import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SeverityBadge, StatusIndicator } from './Status'

describe('status components', () => {
  it('renders semantic severity', () => {
    render(<SeverityBadge severity="CRITICAL" />)
    expect(screen.getByText('CRITICAL')).toHaveClass('critical')
  })

  it('distinguishes degraded collection state', () => {
    render(<StatusIndicator status="degraded" />)
    expect(screen.getByText('degraded')).toHaveClass('degraded')
  })
})
