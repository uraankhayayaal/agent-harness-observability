import path from 'node:path'
import { defineConfig } from '@playwright/test'
import { config, rootEnv } from './helpers/env'

// Backend и frontend поднимаются автоматически, если еще не запущены.
// Инфраструктура (Kafka/Postgres/Neo4j) должна быть запущена заранее: make infra-up
export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  outputDir: 'test-results',
  use: {
    baseURL: config.frontendUrl,
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: 'go -C ../backend run ./cmd/server',
      url: `${config.backendUrl}/healthz`,
      reuseExistingServer: true,
      timeout: 120_000,
      cwd: path.resolve(import.meta.dirname),
      env: rootEnv,
    },
    {
      command: 'npm --prefix ../frontend run dev',
      url: config.frontendUrl,
      reuseExistingServer: true,
      timeout: 120_000,
      cwd: path.resolve(import.meta.dirname),
      env: rootEnv,
    },
  ],
})
