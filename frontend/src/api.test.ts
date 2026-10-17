import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { ApiError, NETWORK_ERROR, getOrders, getSurvey, saveSurvey } from './api.ts'
import type { Order, Survey } from './api.ts'

const survey: Survey = {
  greeting: 'Привет! Помогу оформить заказ торта.',
  questions: [
    { id: 'date', text: 'На какую дату нужен торт?' },
    { id: 'weight', text: 'Какой вес торта?' },
  ],
}

const order: Order = {
  id: 1,
  received_at: '2026-10-16T20:33:07Z',
  tg_user_id: 7,
  username: 'anna',
  answers: [{ question_id: 'date', question_text: 'На какую дату нужен торт?', answer: '20 октября' }],
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json; charset=utf-8' },
  })
}

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  fetchMock.mockReset()
  vi.unstubAllGlobals()
})

// caught returns the error a promise rejects with.
async function caught(p: Promise<unknown>): Promise<ApiError> {
  try {
    await p
  } catch (err) {
    expect(err).toBeInstanceOf(ApiError)
    return err as ApiError
  }
  throw new Error('promise resolved, want ApiError')
}

describe('getSurvey', () => {
  test('200 returns the survey', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, survey))

    await expect(getSurvey()).resolves.toEqual(survey)
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/survey', expect.objectContaining({ method: 'GET' }))
  })

  test('404 means there is no survey yet', async () => {
    fetchMock.mockResolvedValue(jsonResponse(404, { error: 'анкета ещё не загружена' }))

    await expect(getSurvey()).resolves.toBeNull()
  })

  test('500 is an error with the text from core', async () => {
    fetchMock.mockResolvedValue(jsonResponse(500, { error: 'внутренняя ошибка сервера' }))

    const err = await caught(getSurvey())
    expect(err.status).toBe(500)
    expect(err.message).toBe('внутренняя ошибка сервера')
  })
})

describe('saveSurvey', () => {
  test('sends PUT with the survey as JSON', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, survey))

    await expect(saveSurvey(survey)).resolves.toEqual(survey)

    const [path, init] = fetchMock.mock.calls[0]
    expect(path).toBe('/api/v1/survey')
    expect(init?.method).toBe('PUT')
    expect(init?.headers).toEqual({ 'Content-Type': 'application/json' })
    expect(JSON.parse(init?.body as string)).toEqual(survey)
  })

  test('400 carries the field errors from details', async () => {
    const details = [
      { field: 'greeting', message: 'обязательное поле' },
      { field: 'questions[1].id', message: 'id "date" уже есть в анкете' },
      { message: 'неизвестное поле "title"' },
    ]
    fetchMock.mockResolvedValue(jsonResponse(400, { error: 'анкета содержит ошибки', details }))

    const err = await caught(saveSurvey(survey))
    expect(err.status).toBe(400)
    expect(err.message).toBe('анкета содержит ошибки')
    expect(err.details).toEqual(details)
  })

  test('400 without details has an empty list', async () => {
    fetchMock.mockResolvedValue(jsonResponse(400, { error: 'не удалось прочитать тело запроса (максимум 1 МБ)' }))

    const err = await caught(saveSurvey(survey))
    expect(err.message).toBe('не удалось прочитать тело запроса (максимум 1 МБ)')
    expect(err.details).toEqual([])
  })
})

describe('getOrders', () => {
  test('200 returns the orders', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, [order]))

    await expect(getOrders()).resolves.toEqual([order])
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/orders', expect.objectContaining({ method: 'GET' }))
  })

  test('empty list', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, []))

    await expect(getOrders()).resolves.toEqual([])
  })
})

describe('failures', () => {
  test('network error', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))

    const err = await caught(getOrders())
    expect(err.status).toBe(0)
    expect(err.message).toBe(NETWORK_ERROR)
  })

  test('error response that is not JSON', async () => {
    fetchMock.mockResolvedValue(new Response('<html>Internal Server Error</html>', { status: 500 }))

    const err = await caught(getSurvey())
    expect(err.status).toBe(500)
    expect(err.message).toBe('Сервер вернул ошибку (HTTP 500).')
  })

  test('proxy cannot reach core', async () => {
    // This is what the Vite dev server answers when core is not running.
    fetchMock.mockResolvedValue(new Response('', { status: 502, headers: { 'Content-Type': 'text/plain' } }))

    const err = await caught(getOrders())
    expect(err.status).toBe(502)
    expect(err.message).toBe(NETWORK_ERROR)
  })
})
