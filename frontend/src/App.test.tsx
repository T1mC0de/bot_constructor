import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import App from './App.tsx'

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  fetchMock.mockImplementation(async (path) =>
    path === '/api/v1/orders'
      ? jsonResponse(200, [])
      : jsonResponse(404, { error: 'анкета ещё не загружена' }),
  )
})

afterEach(() => {
  fetchMock.mockReset()
  vi.unstubAllGlobals()
})

function ordersRequests() {
  return fetchMock.mock.calls.filter(([path]) => path === '/api/v1/orders').length
}

test('opens on the survey tab', async () => {
  render(<App />)

  expect(screen.getByRole('heading', { name: 'Конструктор ботов' })).toBeInTheDocument()
  expect(screen.getByRole('tab', { name: 'Анкета' })).toHaveAttribute('aria-selected', 'true')
  expect(await screen.findByLabelText('Приветствие')).toBeVisible()
  expect(ordersRequests()).toBe(0)
})

test('switches tabs, keeps survey edits and reloads orders', async () => {
  const user = userEvent.setup()
  render(<App />)
  await user.type(await screen.findByLabelText('Приветствие'), 'Привет')

  await user.click(screen.getByRole('tab', { name: 'Заявки' }))
  expect(await screen.findByText('Заявок пока нет')).toBeInTheDocument()
  expect(screen.getByRole('tab', { name: 'Заявки' })).toHaveAttribute('aria-selected', 'true')
  expect(screen.getByLabelText('Приветствие')).not.toBeVisible()

  await user.click(screen.getByRole('tab', { name: 'Анкета' }))
  expect(screen.getByLabelText('Приветствие')).toHaveValue('Привет')
  expect(screen.queryByText('Заявок пока нет')).not.toBeInTheDocument()

  await user.click(screen.getByRole('tab', { name: 'Заявки' }))
  await screen.findByText('Заявок пока нет')
  expect(ordersRequests()).toBe(2)
})
