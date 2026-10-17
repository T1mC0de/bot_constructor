import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import type { Order } from './api.ts'
import { formatReceivedAt } from './format.ts'
import OrdersList from './OrdersList.tsx'

function makeOrder(id: number, username: string, answers: [string, string][]): Order {
  return {
    id,
    received_at: '2026-10-16T20:33:07Z',
    tg_user_id: 7,
    username,
    answers: answers.map(([text, answer], i) => ({ question_id: `q${i}`, question_text: text, answer })),
  }
}

const first = makeOrder(1, 'anna', [
  ['На какую дату нужен торт?', '20 октября'],
  ['Какие пожелания к оформлению?', 'Надпись «С днём рождения»\nи свечи'],
])
const second = makeOrder(2, '', [['На какую дату нужен торт?', 'завтра']])

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  fetchMock.mockReset()
  vi.unstubAllGlobals()
})

function card(id: number) {
  return screen.getByRole('article', { name: `Заявка №${id}` })
}

test('shows orders, newest first', async () => {
  fetchMock.mockResolvedValue(jsonResponse(200, [first, second]))
  render(<OrdersList />)

  expect(screen.getByText('Загрузка…')).toBeInTheDocument()
  await screen.findByRole('article', { name: 'Заявка №1' })

  expect(screen.getAllByRole('article').map((a) => a.getAttribute('aria-labelledby'))).toEqual([
    'order-2',
    'order-1',
  ])
  expect(screen.getByText('Заявок: 2')).toBeInTheDocument()
  expect(screen.queryByText('Загрузка…')).not.toBeInTheDocument()

  const c = within(card(1))
  expect(c.getByText('@anna')).toBeInTheDocument()
  expect(c.getByText(formatReceivedAt(first.received_at))).toHaveAttribute('dateTime', first.received_at)
  expect(c.getByText('На какую дату нужен торт?')).toBeInTheDocument()
  expect(c.getByText('20 октября')).toBeInTheDocument()
})

test('shows a dash when the buyer has no username', async () => {
  fetchMock.mockResolvedValue(jsonResponse(200, [second]))
  render(<OrdersList />)

  await screen.findByRole('article', { name: 'Заявка №2' })
  expect(within(card(2)).getByText('—')).toBeInTheDocument()
  expect(within(card(2)).queryByText(/@/)).not.toBeInTheDocument()
})

test('keeps line breaks in answers', async () => {
  fetchMock.mockResolvedValue(jsonResponse(200, [first]))
  render(<OrdersList />)

  await screen.findByRole('article', { name: 'Заявка №1' })
  const answer = within(card(1)).getByText(/и свечи/)
  expect(answer.textContent).toBe('Надпись «С днём рождения»\nи свечи')
  expect(answer.tagName).toBe('DD')
})

test('shows that there are no orders yet', async () => {
  fetchMock.mockResolvedValue(jsonResponse(200, []))
  render(<OrdersList />)

  expect(await screen.findByText('Заявок пока нет')).toBeInTheDocument()
  expect(screen.queryByRole('article')).not.toBeInTheDocument()
})

test('shows a load error', async () => {
  fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))
  render(<OrdersList />)

  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Не удалось загрузить заявки: Не удалось связаться с сервером',
  )
  expect(screen.queryByText('Загрузка…')).not.toBeInTheDocument()
  expect(screen.queryByText('Заявок пока нет')).not.toBeInTheDocument()
})

test('refresh loads new orders', async () => {
  fetchMock.mockResolvedValueOnce(jsonResponse(200, [first]))
  render(<OrdersList />)
  await screen.findByRole('article', { name: 'Заявка №1' })

  let respond: (r: Response) => void = () => {}
  fetchMock.mockReturnValueOnce(new Promise((resolve) => (respond = resolve)))
  const refresh = screen.getByRole('button', { name: 'Обновить' })
  await userEvent.click(refresh)

  // The old list stays on screen while the new one is loading.
  expect(refresh).toBeDisabled()
  expect(card(1)).toBeInTheDocument()

  respond(jsonResponse(200, [first, second]))
  expect(await screen.findByRole('article', { name: 'Заявка №2' })).toBeInTheDocument()
  expect(refresh).toBeEnabled()
  expect(fetchMock).toHaveBeenCalledTimes(2)
})

test('refresh after an error clears it', async () => {
  fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'))
  fetchMock.mockResolvedValueOnce(jsonResponse(200, []))
  render(<OrdersList />)
  await screen.findByRole('alert')

  await userEvent.click(screen.getByRole('button', { name: 'Обновить' }))

  expect(await screen.findByText('Заявок пока нет')).toBeInTheDocument()
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
})

describe('formatReceivedAt', () => {
  test('shows the time in the given time zone', () => {
    expect(formatReceivedAt('2026-10-16T20:33:07Z', 'Europe/Moscow')).toMatch(/^16 октября 2026.*23:33$/)
    expect(formatReceivedAt('2026-10-16T20:33:07Z', 'Asia/Vladivostok')).toMatch(/^17 октября 2026.*06:33$/)
  })
})
