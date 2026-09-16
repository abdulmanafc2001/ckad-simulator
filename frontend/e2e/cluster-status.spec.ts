import { expect, test } from '@playwright/test'

// The exam provisions, runs and grades against a live Kubernetes cluster, so
// the home page must show that cluster's state and refuse to start a session
// without it. The down state is mocked so the test never needs to stop the
// developer's own minikube.

test('shows the cluster as ready and allows starting', async ({ page }) => {
  await page.goto('/')

  const banner = page.locator('.cluster-status')
  await expect(banner).toBeVisible()
  await expect(banner).toHaveClass(/up/, { timeout: 20_000 })
  await expect(banner).toContainText('Cluster ready')
  await expect(banner).toContainText('Kubernetes control plane is running')

  await expect(page.getByRole('button', { name: /Start a 2-hour exam session/i })).toBeEnabled()
  await expect(page.getByRole('button', { name: 'Re-check' })).toHaveCount(0)
})

test('blocks the exam and explains how to fix it when the cluster is down', async ({ page }) => {
  await page.route('**/api/v1/cluster/status', (route) =>
    route.fulfill({
      status: 503,
      contentType: 'application/json',
      body: JSON.stringify({
        connected: false,
        detail: 'The connection to the server 127.0.0.1:32771 was refused',
      }),
    }),
  )

  await page.goto('/')

  const banner = page.locator('.cluster-status')
  await expect(banner).toHaveClass(/down/, { timeout: 20_000 })
  await expect(banner).toContainText('Cluster unreachable')
  // The fix must be spelled out, not left to the candidate to guess.
  await expect(banner).toContainText('minikube start')
  // And the underlying reason is surfaced for debugging.
  await expect(banner.locator('.cluster-detail')).toContainText('connection to the server')

  // The exam cannot be started in this state.
  const start = page.getByRole('button', { name: /Start a 2-hour exam session/i })
  await expect(start).toBeDisabled()

  await expect(page.getByRole('button', { name: 'Re-check' })).toBeVisible()
})

test('recovers on its own once the cluster comes back', async ({ page }) => {
  let down = true
  await page.route('**/api/v1/cluster/status', (route) =>
    route.fulfill({
      status: down ? 503 : 200,
      contentType: 'application/json',
      body: JSON.stringify(
        down
          ? { connected: false, detail: 'refused' }
          : { connected: true, detail: 'Kubernetes control plane is running at https://127.0.0.1:32771' },
      ),
    }),
  )

  await page.goto('/')
  await expect(page.locator('.cluster-status')).toHaveClass(/down/, { timeout: 20_000 })
  await expect(page.getByRole('button', { name: /Start a 2-hour exam session/i })).toBeDisabled()

  // Cluster comes back; the page polls every 5s while down.
  down = false
  await expect(page.locator('.cluster-status')).toHaveClass(/up/, { timeout: 20_000 })
  await expect(page.getByRole('button', { name: /Start a 2-hour exam session/i })).toBeEnabled()
})
