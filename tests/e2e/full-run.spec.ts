import { expect, test } from '@playwright/test'
import { config } from '../helpers/env'
import { waitForTrajectory, backendHealthy, frontendReachable } from '../helpers/api'
import {
  demoSteps,
  disconnectProducer,
  newRunID,
  publishRun,
  waitForConsumerGroup,
} from '../helpers/kafka'

test.describe.configure({ mode: 'serial' })

test.beforeAll(async () => {
  expect(
    await backendHealthy(),
    `бэкенд недоступен на ${config.backendUrl} — запустите: make backend`,
  ).toBe(true)
  expect(
    await frontendReachable(),
    `фронтенд недоступен на ${config.frontendUrl} — запустите: make frontend`,
  ).toBe(true)
  // StartOffset=last: публиковать можно только когда конsumer уже в группе,
  // иначе сообщения будут пропущены
  await waitForConsumerGroup(config.kafkaGroupID)
})

test.afterAll(async () => {
  await disconnectProducer()
})

test('полный прогон: сообщение в Kafka → бэкенд → API → визуализация во фронтенде', async ({
  page,
}) => {
  const runID = newRunID()
  const steps = demoSteps(runID)

  // 1. Публикуем шаги в топик (key = run_id)
  await publishRun(steps)

  // 2. Бэкенд вычитывает и сохраняет в обе БД — ждем готовности траектории в API
  const traj = await waitForTrajectory(runID, steps.length)

  expect(traj.nodes).toHaveLength(5)
  expect(traj.nodes.map((n) => n.data.stepNumber)).toEqual([1, 2, 3, 4, 5])
  expect(traj.nodes[3].data.status).toBe('error')
  expect(traj.nodes[1].data.toolCalled).toBe('bash')
  expect(traj.edges).toHaveLength(4)
  // animated: успешный вызов инструмента (step 2 bash) — edge 2→3 и 3→4? edge анимированный, если from-node success + tool
  const animated = traj.edges.filter((e) => e.animated)
  expect(animated.length).toBeGreaterThanOrEqual(1)

  // 3. Открываем фронтенд с run_id — визуальная оболочка
  await page.goto(`${config.frontendUrl}/?run=${runID}`)

  // Заголовок со сводкой
  await expect(page.getByText('5 шагов')).toBeVisible()
  await expect(page.getByText('2200 токенов')).toBeVisible()
  await expect(page.getByText('1 ошибок')).toBeVisible()

  // Граф: 5 узлов и 4 связи на холсте React Flow
  const nodes = page.locator('.react-flow__node')
  await expect(nodes).toHaveCount(5, { timeout: 15_000 })
  await expect(page.locator('.react-flow__edge')).toHaveCount(4)

  // Содержимое узлов
  await expect(page.getByText('Step 1')).toBeVisible()
  await expect(page.getByText('Step 5')).toBeVisible()
  await expect(nodes.filter({ hasText: 'Step 2' }).getByText('bash')).toBeVisible()
  await expect(page.getByText('Inspecting the repository structure.')).toBeVisible()
  await expect(page.getByText('300 tokens')).toBeVisible()

  // Error-шаг: красная рамка + метка error
  const errorNode = nodes.filter({ hasText: 'Step 4' })
  await expect(errorNode.locator('.border-red-500')).toBeVisible()
  await expect(errorNode.getByText('error')).toBeVisible()

  // Success-шаг: синяя рамка
  const okNode = nodes.filter({ hasText: 'Step 1' })
  await expect(okNode.locator('.border-blue-500')).toBeVisible()

  // Скриншот визуальной оболочки как артефакт
  await page.screenshot({ path: test.info().outputPath('trajectory.png'), fullPage: true })
})

test('API возвращает 404 для неизвестного run_id', async () => {
  const res = await fetch(
    `${config.backendUrl}${config.apiBasePath}/runs/00000000-0000-4000-8000-000000000000/trajectory`,
  )
  expect(res.status).toBe(404)
})
