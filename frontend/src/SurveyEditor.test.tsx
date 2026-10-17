import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import type { Survey } from './api.ts'
import SurveyEditor from './SurveyEditor.tsx'

const survey: Survey = {
  greeting: 'Привет! Помогу оформить заказ торта.',
  questions: [
    { id: 'date', text: 'На какую дату нужен торт?' },
    { id: 'weight', text: 'Какой вес торта?' },
    { id: 'design', text: 'Какие пожелания к оформлению?' },
  ],
}

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

// serve answers GET with the given response and PUT with put (by default it
// echoes the body back, as core does for a valid survey).
function serve(get: Response, put?: () => Response | Promise<Response>) {
  fetchMock.mockImplementation(async (_path, init) => {
    if (init?.method === 'PUT') {
      return put ? put() : jsonResponse(200, JSON.parse(init.body as string))
    }
    return get
  })
}

function putCalls() {
  return fetchMock.mock.calls.filter(([, init]) => init?.method === 'PUT')
}

async function renderLoaded() {
  render(<SurveyEditor />)
  return screen.findByRole('list', { name: 'Вопросы' })
}

function questionItems() {
  return within(screen.getByRole('list', { name: 'Вопросы' })).getAllByRole('listitem')
}

function ids() {
  return questionItems().map((li) => (within(li).getByLabelText('id') as HTMLInputElement).value)
}

test('shows the current survey', async () => {
  serve(jsonResponse(200, survey))
  await renderLoaded()

  expect(screen.getByLabelText('Приветствие')).toHaveValue(survey.greeting)
  expect(ids()).toEqual(['date', 'weight', 'design'])
  expect(within(questionItems()[1]).getByLabelText('Текст')).toHaveValue('Какой вес торта?')
  expect(screen.getByText(`${[...survey.greeting].length} / 1000`)).toBeInTheDocument()
})

test('without a survey shows an empty form with one question', async () => {
  serve(jsonResponse(404, { error: 'анкета ещё не загружена' }))
  await renderLoaded()

  expect(screen.getByLabelText('Приветствие')).toHaveValue('')
  expect(ids()).toEqual([''])
  expect(screen.getByText('0 / 1000')).toBeInTheDocument()
})

test('shows a load error and retries', async () => {
  fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'))
  fetchMock.mockResolvedValueOnce(jsonResponse(200, survey))
  render(<SurveyEditor />)

  expect(await screen.findByRole('alert')).toHaveTextContent('Не удалось загрузить анкету')
  await userEvent.click(screen.getByRole('button', { name: 'Повторить' }))

  await screen.findByRole('list', { name: 'Вопросы' })
  expect(ids()).toEqual(['date', 'weight', 'design'])
})

test('adds a question up to 20', async () => {
  const user = userEvent.setup()
  const many = Array.from({ length: 18 }, (_, i) => ({ id: `q${i}`, text: 'Вопрос?' }))
  serve(jsonResponse(200, { greeting: 'Привет', questions: many }))
  await renderLoaded()

  const add = screen.getByRole('button', { name: 'Добавить вопрос' })
  await user.click(add)
  expect(questionItems()).toHaveLength(19)
  expect(ids()[18]).toBe('')
  expect(add).toBeEnabled()

  await user.click(add)
  expect(questionItems()).toHaveLength(20)
  expect(add).toBeDisabled()
})

test('removes a question', async () => {
  serve(jsonResponse(200, survey))
  await renderLoaded()

  await userEvent.click(within(questionItems()[1]).getByRole('button', { name: 'Удалить' }))

  expect(ids()).toEqual(['date', 'design'])
})

test('moves questions up and down', async () => {
  const user = userEvent.setup()
  serve(jsonResponse(200, survey))
  await renderLoaded()

  expect(within(questionItems()[0]).getByRole('button', { name: 'Выше' })).toBeDisabled()
  expect(within(questionItems()[2]).getByRole('button', { name: 'Ниже' })).toBeDisabled()

  await user.click(within(questionItems()[2]).getByRole('button', { name: 'Выше' }))
  expect(ids()).toEqual(['date', 'design', 'weight'])

  await user.click(within(questionItems()[0]).getByRole('button', { name: 'Ниже' }))
  expect(ids()).toEqual(['design', 'date', 'weight'])
})

test('saves the survey', async () => {
  const user = userEvent.setup()
  serve(jsonResponse(404, { error: 'анкета ещё не загружена' }))
  await renderLoaded()

  await user.type(screen.getByLabelText('Приветствие'), 'Привет!')
  await user.type(within(questionItems()[0]).getByLabelText('id'), 'date')
  await user.type(within(questionItems()[0]).getByLabelText('Текст'), 'На какую дату?')
  await user.click(screen.getByRole('button', { name: 'Сохранить' }))

  expect(await screen.findByRole('status')).toHaveTextContent('Анкета сохранена')
  expect(putCalls()).toHaveLength(1)
  expect(JSON.parse(putCalls()[0][1]?.body as string)).toEqual({
    greeting: 'Привет!',
    questions: [{ id: 'date', text: 'На какую дату?' }],
  })
})

test('shows validation errors and does not send the survey', async () => {
  const user = userEvent.setup()
  serve(jsonResponse(200, survey))
  await renderLoaded()

  await user.clear(screen.getByLabelText('Приветствие'))
  const second = within(questionItems()[1]).getByLabelText('id')
  await user.clear(second)
  await user.type(second, 'дата')
  await user.click(screen.getByRole('button', { name: 'Сохранить' }))

  expect(screen.getByRole('alert')).toHaveTextContent('Анкета содержит ошибки')
  expect(screen.getByLabelText('Приветствие')).toHaveAttribute('aria-invalid', 'true')
  expect(screen.getByText('обязательное поле')).toBeInTheDocument()
  expect(within(questionItems()[1]).getByText(/только латиница/)).toBeInTheDocument()
  expect(within(questionItems()[0]).queryByText(/только латиница/)).not.toBeInTheDocument()
  expect(putCalls()).toHaveLength(0)

  // The error moves with its question and disappears once the field is edited.
  await user.click(within(questionItems()[1]).getByRole('button', { name: 'Выше' }))
  expect(within(questionItems()[0]).getByText(/только латиница/)).toBeInTheDocument()
  await user.type(within(questionItems()[0]).getByLabelText('id'), 'x')
  expect(screen.queryByText(/только латиница/)).not.toBeInTheDocument()
})

test('shows an error for the last question removed', async () => {
  serve(jsonResponse(404, { error: 'анкета ещё не загружена' }))
  await renderLoaded()

  await userEvent.type(screen.getByLabelText('Приветствие'), 'Привет')
  await userEvent.click(within(questionItems()[0]).getByRole('button', { name: 'Удалить' }))
  await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))

  expect(screen.getByText('нужен хотя бы один вопрос')).toBeInTheDocument()
  expect(putCalls()).toHaveLength(0)
})

test('shows server errors at their fields', async () => {
  serve(jsonResponse(200, survey), () =>
    jsonResponse(400, {
      error: 'анкета содержит ошибки',
      details: [
        { field: 'questions[1].text', message: 'текст отклонён сервером' },
        { message: 'неизвестное поле "title"' },
      ],
    }),
  )
  await renderLoaded()

  await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))

  const alert = await screen.findByRole('alert')
  expect(alert).toHaveTextContent('Анкета содержит ошибки')
  expect(alert).toHaveTextContent('неизвестное поле "title"')
  expect(within(questionItems()[1]).getByText('текст отклонён сервером')).toBeInTheDocument()
  expect(within(questionItems()[1]).getByLabelText('Текст')).toHaveAttribute('aria-invalid', 'true')
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
})

test('shows a save error that is not about fields', async () => {
  serve(jsonResponse(200, survey), () => jsonResponse(500, { error: 'внутренняя ошибка сервера' }))
  await renderLoaded()

  await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))

  expect(await screen.findByRole('alert')).toHaveTextContent(
    'Не удалось сохранить анкету: внутренняя ошибка сервера',
  )
})

test('the save button is disabled while saving', async () => {
  let respond: (r: Response) => void = () => {}
  serve(jsonResponse(200, survey), () => new Promise((resolve) => (respond = resolve)))
  await renderLoaded()

  await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))

  const button = screen.getByRole('button', { name: 'Сохранение…' })
  expect(button).toBeDisabled()
  expect(screen.getByLabelText('Приветствие')).toBeDisabled()

  respond(jsonResponse(200, survey))
  await waitFor(() => expect(screen.getByRole('button', { name: 'Сохранить' })).toBeEnabled())
  expect(screen.getByRole('status')).toHaveTextContent('Анкета сохранена')
})
