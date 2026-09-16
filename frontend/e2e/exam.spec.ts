import { expect, test } from '@playwright/test'

// End-to-end journey through the real UI: home -> 2-hour exam -> terminal ->
// finish -> results review. Requires the backend on :8080, the Vite dev server
// on :5173 and a live Kubernetes cluster.

test.describe.configure({ mode: 'serial' })

test('home page lists the question bank', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByRole('heading', { name: /Practice the Certified Kubernetes/i })).toBeVisible()

  // Stat cards are populated from GET /api/v1/questions.
  const questionsStat = page.locator('.stat-card').filter({ hasText: 'Questions' })
  await expect(questionsStat.locator('.stat-value')).not.toHaveText('0', { timeout: 15_000 })
  const bankCount = Number(await questionsStat.locator('.stat-value').innerText())
  expect(bankCount).toBeGreaterThan(100)

  await expect(page.locator('.stat-card').filter({ hasText: 'Domains' }).locator('.stat-value')).toHaveText('5')

  // All five CKAD domains render with questions under them.
  for (const d of [
    'Application Design & Build',
    'Application Deployment',
    'Observability & Maintenance',
    'Environment, Config & Security',
    'Services & Networking',
  ]) {
    await expect(page.getByRole('heading', { name: d })).toBeVisible()
  }
  expect(await page.locator('.question-item').count()).toBe(bankCount)
})

test('theme toggle persists across reloads', async ({ page }) => {
  await page.goto('/')
  const html = page.locator('html')
  await expect(html).toHaveAttribute('data-theme', 'dark')
  await page.getByRole('button', { name: 'Toggle color theme' }).click()
  await expect(html).toHaveAttribute('data-theme', 'light')
  await page.reload()
  await expect(html).toHaveAttribute('data-theme', 'light')
  await page.getByRole('button', { name: 'Toggle color theme' }).click()
  await expect(html).toHaveAttribute('data-theme', 'dark')
})

test('full exam journey: start, navigate, hint, terminal, finish, review', async ({ page }) => {
  test.setTimeout(900_000)
  await page.goto('/')

  // --- Start the exam (provisions 17 namespaces on the live cluster) ---
  await page.getByRole('button', { name: /Start a 2-hour exam session/i }).click()
  await expect(page.locator('.exam')).toBeVisible({ timeout: 420_000 })

  // 17 questions, one navigator dot each.
  await expect(page.locator('.q-dot')).toHaveCount(17)
  await expect(page.getByRole('heading', { name: /^Question 1\./ })).toBeVisible()

  // --- Timer is counting down from 2 hours ---
  const timer = page.locator('.timer, [class*="timer"]').first()
  await expect(timer).toBeVisible()
  const t1 = await timer.innerText()
  expect(t1).toMatch(/1:5\d:\d\d|2:00:00|119:|1h/)
  await page.waitForTimeout(2200)
  expect(await timer.innerText()).not.toBe(t1) // it actually ticks

  // --- Question navigation ---
  await page.locator('.q-dot').nth(4).click()
  await expect(page.getByRole('heading', { name: /^Question 5\./ })).toBeVisible()
  await page.getByRole('button', { name: /Previous/ }).click()
  await expect(page.getByRole('heading', { name: /^Question 4\./ })).toBeVisible()
  await page.getByRole('button', { name: /Next/ }).click()
  await expect(page.getByRole('heading', { name: /^Question 5\./ })).toBeVisible()

  // Jump-to dropdown marks viewed questions with a check.
  await expect(page.locator('.q-select option').nth(0)).toContainText('✓')

  // --- Flag for review ---
  await page.getByRole('button', { name: /Flag for review/ }).click()
  await expect(page.getByRole('button', { name: /Flagged for review/ })).toBeVisible()
  await expect(page.locator('.exam-progress')).toContainText('1 flagged')
  await expect(page.locator('.q-dot.flagged')).toHaveCount(1)

  // --- Hints reveal one at a time ---
  await expect(page.locator('.q-hints')).toContainText('No hints revealed')
  await page.getByRole('button', { name: /Reveal hint \(0\// }).click()
  await expect(page.locator('.hint-list li')).toHaveCount(1)

  // --- Grading and solutions must NOT be rendered during the exam ---
  await expect(page.locator('.results')).toHaveCount(0)
  await expect(page.locator('.check-list')).toHaveCount(0)
  await expect(page.locator('.review-item')).toHaveCount(0)
  await expect(page.locator('.results-pct')).toHaveCount(0)

  // --- The embedded terminal runs real kubectl against the cluster ---
  const term = page.locator('.xterm-screen, .xterm').first()
  await expect(term).toBeVisible()
  await term.click()
  // Keep output to a line or two: xterm virtualizes rows, so long output
  // scrolls out of the DOM and can't be asserted on.
  await page.keyboard.type('kubectl get ns default -o name')
  await page.keyboard.press('Enter')
  await expect(page.locator('.xterm')).toContainText('namespace/default', { timeout: 30_000 })

  // Terminal supports pipes.
  await page.keyboard.type('kubectl get ns -o name | grep kube-system')
  await page.keyboard.press('Enter')
  await expect(page.locator('.xterm')).toContainText('namespace/kube-system', { timeout: 30_000 })

  // Errors surface in the terminal rather than breaking the UI.
  await page.keyboard.type('kubectl get nosuchresource')
  await page.keyboard.press('Enter')
  await expect(page.locator('.xterm')).toContainText(/error|Error/, { timeout: 30_000 })

  // Built-in editor: `vi` is intercepted client-side, not sent to the backend.
  await page.keyboard.type('vi notes.txt')
  await page.keyboard.press('Enter')
  await page.waitForTimeout(1500)
  await expect(page.locator('.xterm')).not.toContainText('command not allowed')

  // --- Finish the exam (grades all 17 questions against the live cluster) ---
  // Ending the session is irreversible, so it is confirmed first.
  await page.getByRole('button', { name: /Finish exam/ }).click()
  const confirm = page.getByRole('dialog', { name: 'Finish the exam?' })
  await expect(confirm).toBeVisible()
  // Backing out returns to the exam with nothing graded.
  await confirm.getByRole('button', { name: 'Keep working' }).click()
  await expect(confirm).toHaveCount(0)
  await expect(page.locator('.exam')).toBeVisible()

  await page.getByRole('button', { name: /Finish exam/ }).click()
  await page.getByRole('button', { name: /Finish & grade/ }).click()
  // Grading takes minutes against a live cluster, so progress is shown.
  await expect(page.locator('.grading-overlay')).toBeVisible()
  await expect(page.locator('.results')).toBeVisible({ timeout: 300_000 })

  // --- Results: score, verdict, one review row per question ---
  await expect(page.locator('.results-pct')).toHaveText(/^\d+%$/)
  await expect(page.locator('.results-summary')).toHaveClass(/pass|fail/)
  await expect(page.locator('.review-item')).toHaveCount(17)

  // Failed questions auto-expand so the candidate sees what went wrong.
  const failed = page.locator('.review-item.bad').first()
  await expect(failed.locator('.review-body')).toBeVisible()
  await expect(failed.locator('.check-list .check').first()).toBeVisible()
  await expect(failed.locator('.review-body')).toContainText('Check results')
  await expect(failed.locator('.review-body')).toContainText('How to do it correctly')
  await expect(failed.locator('.review-body pre')).not.toBeEmpty()
  // Failed checks show the observed cluster output that caused the miss.
  await expect(failed.locator('.check.bad').first()).toBeVisible()

  // The row header toggles the body closed and open again.
  await failed.locator('.review-head').click()
  await expect(failed.locator('.review-body')).toHaveCount(0)
  await failed.locator('.review-head').click()
  await expect(failed.locator('.review-body')).toBeVisible()

  // Every row shows an earned/max score.
  for (const s of await page.locator('.review-score').allInnerTexts()) {
    expect(s).toMatch(/\d+\s*\/\s*\d+/)
  }

  // Back to home.
  await page.locator('.brand').click()
  await expect(page.getByRole('button', { name: /Start a 2-hour exam session/i })).toBeVisible()
})
