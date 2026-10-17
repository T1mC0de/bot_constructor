import { describe, expect, test } from 'vitest'
import type { FieldError, Question, Survey } from './api.ts'
import { charCount, validateSurvey } from './validation.ts'

function validSurvey(): Survey {
  return {
    greeting: 'Привет! Помогу оформить заказ торта.',
    questions: [
      { id: 'date', text: 'На какую дату нужен торт?' },
      { id: 'weight', text: 'Какой вес торта?' },
    ],
  }
}

function manyQuestions(n: number): Question[] {
  return Array.from({ length: n }, (_, i) => ({ id: `q${i}`, text: 'Вопрос?' }))
}

// fields returns the fields of the errors, "" for errors without a field.
function fields(errs: FieldError[]): string {
  return errs.map((e) => e.field ?? '').join(',')
}

describe('validateSurvey', () => {
  // want is the comma-separated fields with errors, as in the core tests.
  const cases: [name: string, change: (s: Survey) => void, want: string][] = [
    ['valid', () => {}, ''],
    ['no greeting', (s) => (s.greeting = ''), 'greeting'],
    ['greeting of spaces', (s) => (s.greeting = '  \n '), 'greeting'],
    ['greeting 1000 chars', (s) => (s.greeting = 'я'.repeat(1000)), ''],
    ['greeting 1001 chars', (s) => (s.greeting = 'я'.repeat(1001)), 'greeting'],
    ['greeting 1000 chars and spaces around', (s) => (s.greeting = `  ${'я'.repeat(1000)}\n`), ''],
    ['greeting 1000 emoji', (s) => (s.greeting = '🎂'.repeat(1000)), ''],
    ['greeting 1001 emoji', (s) => (s.greeting = '🎂'.repeat(1001)), 'greeting'],
    ['no questions', (s) => (s.questions = []), 'questions'],
    ['20 questions', (s) => (s.questions = manyQuestions(20)), ''],
    ['21 questions', (s) => (s.questions = manyQuestions(21)), 'questions'],
    ['no id', (s) => (s.questions[0].id = ''), 'questions[0].id'],
    ['id allowed chars', (s) => (s.questions[0].id = 'Ab_9-z'), ''],
    ['id 32 chars', (s) => (s.questions[0].id = 'a'.repeat(32)), ''],
    ['id 33 chars', (s) => (s.questions[0].id = 'a'.repeat(33)), 'questions[0].id'],
    ['id in cyrillic', (s) => (s.questions[0].id = 'дата'), 'questions[0].id'],
    ['id with space', (s) => (s.questions[0].id = 'my date'), 'questions[0].id'],
    ['id with emoji', (s) => (s.questions[0].id = 'cake🎂'), 'questions[0].id'],
    ['duplicate id', (s) => (s.questions[1].id = 'date'), 'questions[1].id'],
    ['no text', (s) => (s.questions[1].text = ' '), 'questions[1].text'],
    ['text 300 chars', (s) => (s.questions[0].text = 'ж'.repeat(300)), ''],
    ['text 301 chars', (s) => (s.questions[0].text = 'ж'.repeat(301)), 'questions[0].text'],
    ['text 300 emoji', (s) => (s.questions[0].text = '🍰'.repeat(300)), ''],
    ['text 301 emoji', (s) => (s.questions[0].text = '🍰'.repeat(301)), 'questions[0].text'],
    [
      'several errors at once',
      (s) => {
        s.greeting = ''
        s.questions = [...manyQuestions(20), { id: 'q0', text: '' }]
      },
      'greeting,questions,questions[20].id,questions[20].text',
    ],
  ]

  test.each(cases)('%s', (_, change, want) => {
    const s = validSurvey()
    change(s)
    expect(fields(validateSurvey(s))).toBe(want)
  })

  test('messages are the same as in core', () => {
    const errs = validateSurvey({
      greeting: 'я'.repeat(1001),
      questions: [
        { id: 'date', text: '' },
        { id: 'date', text: 'ж'.repeat(301) },
        { id: 'оформление', text: 'Пожелания?' },
      ],
    })
    expect(errs).toEqual([
      { field: 'greeting', message: 'не длиннее 1000 символов, сейчас 1001' },
      { field: 'questions[0].text', message: 'обязательное поле' },
      { field: 'questions[1].id', message: 'id "date" уже есть в анкете' },
      { field: 'questions[1].text', message: 'не длиннее 300 символов, сейчас 301' },
      { field: 'questions[2].id', message: 'только латиница, цифры, «_» и «-», не длиннее 32 символов' },
    ])
    expect(validateSurvey({ ...validSurvey(), questions: manyQuestions(21) })).toEqual([
      { field: 'questions', message: 'не больше 20 вопросов, сейчас 21' },
    ])
    expect(validateSurvey({ ...validSurvey(), questions: [] })).toEqual([
      { field: 'questions', message: 'нужен хотя бы один вопрос' },
    ])
  })
})

describe('charCount', () => {
  test.each([
    ['', 0],
    ['торт', 4],
    ['🎂', 1],
    ['торт 🎂🍰', 7],
  ])('%j has %i characters', (s, want) => {
    expect(charCount(s)).toBe(want)
  })
})
