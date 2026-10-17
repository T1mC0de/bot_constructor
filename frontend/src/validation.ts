// Survey validation in the browser. The rules, field paths and messages are the
// same as in core (backend/core/internal/model/survey.go), so the seller sees
// the same errors whether they come from here or from the server.

import type { FieldError, Survey } from './api.ts'

export const MAX_GREETING_LEN = 1000
export const MAX_QUESTIONS = 20
export const MAX_QUESTION_TEXT_LEN = 300

const QUESTION_ID_PATTERN = /^[A-Za-z0-9_-]{1,32}$/

// charCount counts Unicode characters, as core does. str.length counts UTF-16
// code units, so an emoji would count as two.
export function charCount(s: string): number {
  return [...s].length
}

// validateSurvey returns every problem in the survey, not just the first one.
export function validateSurvey(s: Survey): FieldError[] {
  const errs: FieldError[] = []
  const add = (field: string, message: string) => errs.push({ field, message })

  const greetingLen = charCount(s.greeting.trim())
  if (greetingLen === 0) {
    add('greeting', 'обязательное поле')
  } else if (greetingLen > MAX_GREETING_LEN) {
    add('greeting', `не длиннее ${MAX_GREETING_LEN} символов, сейчас ${greetingLen}`)
  }

  const n = s.questions.length
  if (n === 0) {
    add('questions', 'нужен хотя бы один вопрос')
  } else if (n > MAX_QUESTIONS) {
    add('questions', `не больше ${MAX_QUESTIONS} вопросов, сейчас ${n}`)
  }

  const seen = new Set<string>()
  s.questions.forEach((q, i) => {
    if (q.id === '') {
      add(`questions[${i}].id`, 'обязательное поле')
    } else if (!QUESTION_ID_PATTERN.test(q.id)) {
      add(`questions[${i}].id`, 'только латиница, цифры, «_» и «-», не длиннее 32 символов')
    } else if (seen.has(q.id)) {
      add(`questions[${i}].id`, `id ${JSON.stringify(q.id)} уже есть в анкете`)
    }
    seen.add(q.id)

    const textLen = charCount(q.text.trim())
    if (textLen === 0) {
      add(`questions[${i}].text`, 'обязательное поле')
    } else if (textLen > MAX_QUESTION_TEXT_LEN) {
      add(`questions[${i}].text`, `не длиннее ${MAX_QUESTION_TEXT_LEN} символов, сейчас ${textLen}`)
    }
  })
  return errs
}
