// Client for the core HTTP API. Paths are relative: in development the Vite
// dev server proxies /api to core (see vite.config.ts).

export interface Question {
  id: string
  text: string
}

export interface Survey {
  greeting: string
  questions: Question[]
}

export interface Answer {
  question_id: string
  question_text: string
  answer: string
}

export interface Order {
  id: number
  received_at: string // RFC 3339, UTC
  tg_user_id: number
  username: string // without "@"; may be empty
  answers: Answer[]
}

// FieldError is one problem in a survey. field is a path like "greeting" or
// "questions[2].id"; it is absent when the problem concerns the whole survey.
export interface FieldError {
  field?: string
  message: string
}

// ApiError is any failed request. status is 0 when core could not be reached.
export class ApiError extends Error {
  readonly status: number
  readonly details: FieldError[]

  constructor(status: number, message: string, details: FieldError[] = []) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.details = details
  }
}

// errorMessage is the text to show for an error caught from the functions below.
export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

export const NETWORK_ERROR = 'Не удалось связаться с сервером. Проверьте, что core запущен.'

const SURVEY_PATH = '/api/v1/survey'
const ORDERS_PATH = '/api/v1/orders'

// getSurvey returns the current survey or null if none has been uploaded yet.
export async function getSurvey(): Promise<Survey | null> {
  try {
    return await request<Survey>('GET', SURVEY_PATH)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      return null
    }
    throw err
  }
}

// saveSurvey replaces the survey and returns it as core stored it (texts trimmed).
// On validation errors it throws an ApiError with details.
export function saveSurvey(survey: Survey): Promise<Survey> {
  return request<Survey>('PUT', SURVEY_PATH, survey)
}

export async function getOrders(): Promise<Order[]> {
  return (await request<Order[] | null>('GET', ORDERS_PATH)) ?? []
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response
  try {
    res = await fetch(path, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, NETWORK_ERROR)
  }

  // Error responses of core are JSON, but a proxy in between may answer with
  // an empty or HTML body, so parsing must not fail the whole request.
  const data: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    throw errorFromResponse(res.status, data)
  }
  return data as T
}

function errorFromResponse(status: number, data: unknown): ApiError {
  if (!isRecord(data) || typeof data.error !== 'string') {
    // The Vite proxy answers 502 in plain text when core is not running.
    const unreachable = status === 502 || status === 503 || status === 504
    return new ApiError(status, unreachable ? NETWORK_ERROR : `Сервер вернул ошибку (HTTP ${status}).`)
  }
  const details = Array.isArray(data.details) ? data.details.filter(isFieldError) : []
  return new ApiError(status, data.error, details)
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null
}

function isFieldError(v: unknown): v is FieldError {
  return (
    isRecord(v) &&
    typeof v.message === 'string' &&
    (v.field === undefined || typeof v.field === 'string')
  )
}
