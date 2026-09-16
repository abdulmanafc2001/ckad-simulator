import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

// Exams are persisted server-side, so the app asks GET /api/v1/sessions on
// load: an exam still running is resumed, and finished ones are listed for
// review. Both responses are mocked here — driving the real thing would mean
// starting and abandoning a two-hour session against the live cluster.

const HOUR_NS = 3_600_000_000_000

/** A question complete enough for the exam view to render. */
const question = {
  id: 'q-resume',
  domain: 'application-design',
  difficulty: 'easy',
  title: 'Run a pod',
  description: 'Create a single pod.',
  task: 'Create a pod named web using the nginx image in namespace q-resume.',
  hints: ['kubectl run web --image=nginx -n q-resume'],
  solution: 'kubectl run web --image=nginx -n q-resume',
  prepare: [],
  checks: [{ id: 'c1', description: 'pod exists', weight: 5, command: 'get pod web -n q-resume' }],
  cleanup: [],
  weight: 5,
}

const pastResults = {
  id: 'past-1',
  startedAt: '2026-09-15T09:00:00Z',
  endedAt: '2026-09-15T10:35:00Z',
  earned: 4,
  max: 5,
  totalQuestions: 1,
  passed: true,
  passScore: 66,
  attempts: [
    {
      attemptId: 'a1',
      questionId: 'q-resume',
      question: 'Run a pod',
      task: question.task,
      domain: 'application-design',
      difficulty: 'easy',
      isCorrect: false,
      score: 4,
      maxScore: 5,
      solution: question.solution,
      checks: [
        { checkId: 'c1', description: 'pod exists', passed: true, points: 4, maxPoints: 4 },
        {
          checkId: 'c2',
          description: 'image is nginx',
          passed: false,
          points: 0,
          maxPoints: 1,
          output: 'not found',
        },
      ],
    },
  ],
}

/** Serves the question bank and the single question the mocks refer to. */
async function stubQuestions(page: Page) {
  await page.route('**/api/v1/questions', (route) =>
    route.fulfill({ json: { questions: [question] } }),
  )
  await page.route('**/api/v1/questions/*', (route) => route.fulfill({ json: question }))
}

test('resumes an exam that was still running', async ({ page }) => {
  // Started 40 minutes ago, so ~1h20m should remain on a 2-hour limit.
  const startedAt = new Date(Date.now() - 40 * 60_000).toISOString()
  await stubQuestions(page)
  await page.route('**/api/v1/sessions', (route) =>
    route.fulfill({
      json: {
        active: {
          id: 'live-session',
          questionIds: ['q-resume'],
          startedAt,
          durationLimit: 2 * HOUR_NS,
        },
        history: [],
      },
    }),
  )

  await page.goto('/')

  // Straight into the exam — the home screen must not be offered.
  await expect(page.locator('.exam')).toBeVisible()
  await expect(page.getByRole('button', { name: /Start a 2-hour exam session/i })).toHaveCount(0)
  await expect(page.getByRole('heading', { name: /^Question 1\./ })).toBeVisible()
  await expect(page.locator('.exam-chip')).toContainText('Exam in progress')

  // The timer continues from the original start rather than granting a
  // fresh two hours — that is the whole point of resuming server-side.
  const timer = page.locator('.timer').first()
  await expect(timer).toBeVisible()
  await expect(timer).toHaveText(/^01:(19|18):\d\d$/)
})

test('lists finished exams and reopens their marks', async ({ page }) => {
  await stubQuestions(page)
  await page.route('**/api/v1/sessions', (route) =>
    route.fulfill({
      json: {
        active: null,
        history: [
          {
            id: 'past-1',
            startedAt: '2026-09-15T09:00:00Z',
            endedAt: '2026-09-15T10:35:00Z',
            earned: 4,
            max: 5,
            totalQuestions: 1,
            passed: true,
          },
          {
            id: 'past-2',
            startedAt: '2026-09-14T09:00:00Z',
            endedAt: '2026-09-14T11:00:00Z',
            earned: 1,
            max: 5,
            totalQuestions: 1,
            passed: false,
          },
        ],
      },
    }),
  )
  await page.route('**/api/v1/sessions/past-1/results', (route) =>
    route.fulfill({ json: pastResults }),
  )

  await page.goto('/')

  // No exam in progress, so the home screen with the history is shown.
  await expect(page.getByRole('button', { name: /Start a 2-hour exam session/i })).toBeVisible()
  const rows = page.locator('.history-row')
  await expect(rows).toHaveCount(2)
  await expect(rows.first()).toContainText('Pass')
  await expect(rows.first()).toContainText('80%')
  await expect(rows.first()).toContainText('4/5 points')
  await expect(rows.nth(1)).toContainText('Fail')
  await expect(rows.nth(1)).toContainText('20%')

  // Opening one replays the marks it was given, solutions included.
  await rows.first().click()
  await expect(page.locator('.results')).toBeVisible()
  await expect(page.locator('.results-archived')).toContainText('Reviewing a past exam')
  await expect(page.locator('.results-pct')).toHaveText('80%')
  await expect(page.locator('.review-item')).toHaveCount(1)

  const item = page.locator('.review-item').first()
  await expect(item.locator('.check')).toHaveCount(2)
  await expect(item.locator('.check.bad')).toContainText('image is nginx')

  // And it is clearly a review, not a session that can be continued.
  await page.getByRole('button', { name: 'Back to home' }).click()
  await expect(page.getByRole('button', { name: /Start a 2-hour exam session/i })).toBeVisible()
})
