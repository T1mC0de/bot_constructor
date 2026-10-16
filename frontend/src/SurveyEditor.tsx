import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { ApiError, getSurvey, saveSurvey } from './api.ts'
import type { FieldError, Survey } from './api.ts'
import { MAX_GREETING_LEN, MAX_QUESTIONS, charCount, validateSurvey } from './validation.ts'

// QuestionDraft is a question being edited. key identifies it while questions
// are moved around, so React keeps the inputs and the errors stay with it.
interface QuestionDraft {
  key: number
  id: string
  text: string
}

let lastKey = 0

function newDraft(id = '', text = ''): QuestionDraft {
  lastKey += 1
  return { key: lastKey, id, text }
}

function draftsOf(survey: Survey | null): QuestionDraft[] {
  if (survey === null || survey.questions.length === 0) {
    return [newDraft()]
  }
  return survey.questions.map((q) => newDraft(q.id, q.text))
}

// Errors are the field errors of the last save attempt, grouped by where they
// are shown. Errors of a question are keyed by QuestionDraft.key, not by index.
interface Errors {
  general: string[]
  greeting: string[]
  questions: string[]
  id: Record<number, string[]>
  text: Record<number, string[]>
}

const noErrors: Errors = { general: [], greeting: [], questions: [], id: {}, text: {} }

const QUESTION_FIELD = /^questions\[(\d+)\]\.(id|text)$/

function groupErrors(list: FieldError[], questions: QuestionDraft[]): Errors {
  const errs: Errors = { general: [], greeting: [], questions: [], id: {}, text: {} }
  for (const { field, message } of list) {
    const m = field === undefined ? null : QUESTION_FIELD.exec(field)
    const q = m === null ? undefined : questions[Number(m[1])]
    if (field === 'greeting' || field === 'questions') {
      errs[field].push(message)
    } else if (m !== null && q !== undefined) {
      const byKey = errs[m[2] as 'id' | 'text']
      byKey[q.key] = [...(byKey[q.key] ?? []), message]
    } else {
      errs.general.push(field === undefined ? message : `${field}: ${message}`)
    }
  }
  return errs
}

const INVALID_FORM = 'Анкета содержит ошибки — исправьте отмеченные поля.'

function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

type LoadState = { state: 'loading' } | { state: 'ready' } | { state: 'error'; message: string }

export default function SurveyEditor() {
  const [load, setLoad] = useState<LoadState>({ state: 'loading' })
  const [attempt, setAttempt] = useState(0)
  const [greeting, setGreeting] = useState('')
  const [questions, setQuestions] = useState<QuestionDraft[]>([])
  const [errors, setErrors] = useState<Errors>(noErrors)
  const [formError, setFormError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let ignore = false
    getSurvey().then(
      (survey) => {
        if (ignore) return
        setGreeting(survey?.greeting ?? '')
        setQuestions(draftsOf(survey))
        setLoad({ state: 'ready' })
      },
      (err: unknown) => {
        if (!ignore) setLoad({ state: 'error', message: errorText(err) })
      },
    )
    return () => {
      ignore = true
    }
  }, [attempt])

  function retry() {
    setLoad({ state: 'loading' })
    setAttempt((n) => n + 1)
  }

  function changeGreeting(value: string) {
    setGreeting(value)
    setErrors((e) => ({ ...e, greeting: [] }))
    setNotice(null)
  }

  function changeQuestion(key: number, field: 'id' | 'text', value: string) {
    setQuestions((qs) => qs.map((q) => (q.key === key ? { ...q, [field]: value } : q)))
    setErrors((e) => ({ ...e, [field]: { ...e[field], [key]: [] } }))
    setNotice(null)
  }

  // reorder replaces the list after adding, removing or moving a question.
  // The error about the number of questions may no longer hold, so it goes.
  function reorder(next: QuestionDraft[]) {
    setQuestions(next)
    setErrors((e) => ({ ...e, questions: [] }))
    setNotice(null)
  }

  function move(index: number, delta: -1 | 1) {
    const next = [...questions]
    ;[next[index], next[index + delta]] = [next[index + delta], next[index]]
    reorder(next)
  }

  function showErrors(list: FieldError[]) {
    setErrors(groupErrors(list, questions))
    setFormError(INVALID_FORM)
  }

  async function save(e: FormEvent) {
    e.preventDefault()
    const survey: Survey = { greeting, questions: questions.map(({ id, text }) => ({ id, text })) }
    setNotice(null)

    const invalid = validateSurvey(survey)
    if (invalid.length > 0) {
      showErrors(invalid)
      return
    }

    setSaving(true)
    try {
      const saved = await saveSurvey(survey)
      setGreeting(saved.greeting)
      setQuestions(draftsOf(saved))
      setErrors(noErrors)
      setFormError(null)
      setNotice('Анкета сохранена')
    } catch (err) {
      if (err instanceof ApiError && err.details.length > 0) {
        showErrors(err.details)
      } else {
        setErrors(noErrors)
        setFormError(`Не удалось сохранить анкету: ${errorText(err)}`)
      }
    } finally {
      setSaving(false)
    }
  }

  if (load.state === 'loading') {
    return <p className="status">Загрузка…</p>
  }
  if (load.state === 'error') {
    return (
      <div className="status">
        <p className="error" role="alert">
          Не удалось загрузить анкету: {load.message}
        </p>
        <button type="button" onClick={retry}>
          Повторить
        </button>
      </div>
    )
  }

  const greetingLen = charCount(greeting)

  return (
    <form className="survey" onSubmit={save}>
      {formError !== null && (
        <div className="form-error" role="alert">
          <p>{formError}</p>
          {errors.general.length > 0 && (
            <ul>
              {errors.general.map((msg, i) => (
                <li key={i}>{msg}</li>
              ))}
            </ul>
          )}
        </div>
      )}

      <fieldset disabled={saving}>
        <div className="field">
          <label htmlFor="greeting">Приветствие</label>
          <textarea
            id="greeting"
            rows={4}
            value={greeting}
            aria-invalid={errors.greeting.length > 0}
            onChange={(e) => changeGreeting(e.target.value)}
          />
          <span className={greetingLen > MAX_GREETING_LEN ? 'counter over' : 'counter'}>
            {greetingLen} / {MAX_GREETING_LEN}
          </span>
          <FieldErrors messages={errors.greeting} />
        </div>

        <h2 id="questions-title">Вопросы</h2>
        <p className="hint">Бот задаёт вопросы по порядку. id — латиница, цифры, «_» и «-».</p>
        <FieldErrors messages={errors.questions} />

        <ol className="questions" aria-labelledby="questions-title">
          {questions.map((q, i) => {
            const idErrors = errors.id[q.key] ?? []
            const textErrors = errors.text[q.key] ?? []
            return (
              <li key={q.key} className="question">
                <div className="question-fields">
                  <div className="field field-id">
                    <label htmlFor={`q${q.key}-id`}>id</label>
                    <input
                      id={`q${q.key}-id`}
                      value={q.id}
                      aria-invalid={idErrors.length > 0}
                      onChange={(e) => changeQuestion(q.key, 'id', e.target.value)}
                    />
                    <FieldErrors messages={idErrors} />
                  </div>
                  <div className="field field-text">
                    <label htmlFor={`q${q.key}-text`}>Текст</label>
                    <input
                      id={`q${q.key}-text`}
                      value={q.text}
                      aria-invalid={textErrors.length > 0}
                      onChange={(e) => changeQuestion(q.key, 'text', e.target.value)}
                    />
                    <FieldErrors messages={textErrors} />
                  </div>
                </div>
                <div className="question-actions">
                  <button
                    type="button"
                    aria-label="Выше"
                    title="Выше"
                    disabled={i === 0}
                    onClick={() => move(i, -1)}
                  >
                    ↑
                  </button>
                  <button
                    type="button"
                    aria-label="Ниже"
                    title="Ниже"
                    disabled={i === questions.length - 1}
                    onClick={() => move(i, 1)}
                  >
                    ↓
                  </button>
                  <button
                    type="button"
                    onClick={() => reorder(questions.filter((other) => other.key !== q.key))}
                  >
                    Удалить
                  </button>
                </div>
              </li>
            )
          })}
        </ol>

        <div className="form-actions">
          <button
            type="button"
            disabled={questions.length >= MAX_QUESTIONS}
            onClick={() => reorder([...questions, newDraft()])}
          >
            Добавить вопрос
          </button>
          <span className="hint">
            {questions.length} / {MAX_QUESTIONS}
          </span>
          <button type="submit" className="primary">
            {saving ? 'Сохранение…' : 'Сохранить'}
          </button>
        </div>
      </fieldset>

      {notice !== null && (
        <p className="notice" role="status">
          {notice}
        </p>
      )}
    </form>
  )
}

function FieldErrors({ messages }: { messages: string[] }) {
  return messages.map((msg, i) => (
    <p key={i} className="field-error">
      {msg}
    </p>
  ))
}
