import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ApiError } from '@shared/api/client'
import LoginPage from './LoginPage.vue'

const loginMock = vi.fn()
vi.mock('@modules/auth/store', () => ({
  useAuthStore: () => ({ login: loginMock }),
}))

const replaceMock = vi.fn()
let routeQuery: Record<string, unknown> = {}
vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: replaceMock }),
  useRoute: () => ({ query: routeQuery }),
}))

async function submitPassword(wrapper: ReturnType<typeof mount>, value: string) {
  await wrapper.find('input[type="password"]').setValue(value)
  await wrapper.find('form').trigger('submit')
  await flushPromises()
}

describe('LoginPage', () => {
  beforeEach(() => {
    loginMock.mockReset()
    replaceMock.mockReset()
    routeQuery = {}
  })

  it('submits the password to auth.login', async () => {
    loginMock.mockResolvedValue(undefined)
    const wrapper = mount(LoginPage)

    await submitPassword(wrapper, 'hunter2')

    expect(loginMock).toHaveBeenCalledWith('hunter2')
  })

  it('renders the mapped 401 error and does not navigate on rejected login', async () => {
    loginMock.mockRejectedValue(new ApiError(401, 'Unauthorized'))
    const wrapper = mount(LoginPage)

    await submitPassword(wrapper, 'wrong')

    expect(wrapper.text()).toContain('invalid password')
    expect(replaceMock).not.toHaveBeenCalled()
  })

  it('maps a 429 rejection to the rate-limit message', async () => {
    loginMock.mockRejectedValue(new ApiError(429, 'Too Many Requests'))
    const wrapper = mount(LoginPage)

    await submitPassword(wrapper, 'wrong')

    expect(wrapper.text()).toContain('too many attempts, wait a minute')
  })

  it('navigates via router.replace with the sanitized redirect target on success', async () => {
    loginMock.mockResolvedValue(undefined)
    routeQuery = { redirect: '/dashboard' }
    const wrapper = mount(LoginPage)

    await submitPassword(wrapper, 'hunter2')

    expect(replaceMock).toHaveBeenCalledWith('/dashboard')
  })
})
