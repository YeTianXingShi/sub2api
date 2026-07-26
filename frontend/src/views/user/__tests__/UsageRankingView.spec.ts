import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UsageRankingView from '../UsageRankingView.vue'

const getRanking = vi.hoisted(() => vi.fn())

vi.mock('@/api/usage', () => ({ usageAPI: { getRanking } }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('@/composables/useBalanceDisplay', () => ({
  useBalanceDisplay: () => ({
    balanceUnitName: { value: 'USD' },
    formatBalanceAmount: (amount: number) => `$${amount.toFixed(2)}`,
  }),
}))

const ranking = [
  { rank: 1, user_id: 1, display_name: 'Alpha', avatar_url: '', requests: 10, input_tokens: 100, output_tokens: 50, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 150, actual_cost: 12.5 },
  { rank: 2, user_id: 2, display_name: 'b***a@example.com', avatar_url: '', requests: 9, input_tokens: 80, output_tokens: 40, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 120, actual_cost: 8.25 },
  { rank: 3, user_id: 3, display_name: 'Gamma', avatar_url: 'https://example.test/avatar.png', requests: 8, input_tokens: 70, output_tokens: 30, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 100, actual_cost: 4 },
]

function mountView() {
  return mount(UsageRankingView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        DateRangePicker: { template: '<div data-testid="date-range" />' },
        LoadingSpinner: { template: '<span />' },
        Icon: { template: '<span />' },
      },
    },
  })
}

describe('UsageRankingView', () => {
  beforeEach(() => {
    getRanking.mockReset()
  })

  it('highlights the top three and uses actual cost as the primary metric', async () => {
    getRanking.mockResolvedValue({
      ranking,
      total_requests: 27,
      total_tokens: 370,
      total_actual_cost: 24.75,
      start_date: '2026-07-25',
      end_date: '2026-07-25',
      limit: 20,
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.findAll('[data-testid="usage-ranking-top-card"]')).toHaveLength(3)
    expect(wrapper.findAll('[data-testid="usage-ranking-row"]')).toHaveLength(3)
    expect(wrapper.findAll('[data-testid="usage-ranking-primary-cost"]').map((node) => node.text())).toEqual(['$12.50', '$8.25', '$4.00'])
    expect(wrapper.text()).toContain('b***a@example.com')
    expect(getRanking).toHaveBeenCalledWith(expect.objectContaining({
      start_date: expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/),
      end_date: expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/),
      timezone: expect.any(String),
    }))
  })

  it('renders the empty state when there is no positive spending', async () => {
    getRanking.mockResolvedValue({ ranking: [], total_requests: 0, total_tokens: 0, total_actual_cost: 0, start_date: '2026-07-25', end_date: '2026-07-25', limit: 20 })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('usageRanking.emptyTitle')
    expect(wrapper.findAll('[data-testid="usage-ranking-row"]')).toHaveLength(0)
  })
})
