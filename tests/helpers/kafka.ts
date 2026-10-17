import { randomUUID } from 'node:crypto'
import { Kafka, logLevel, type Producer } from 'kafkajs'
import { config } from './env'

export interface StepMessage {
  run_id: string
  harness_name: string
  agent_version: string
  step_number: number
  thought: string
  tool_called?: string
  status: 'success' | 'error'
  latency_ms: number
  tokens_used: number
  cost_usd: number
}

let producer: Producer | null = null

async function getProducer(): Promise<Producer> {
  if (producer) return producer
  const kafka = new Kafka({
    clientId: 'e2e-tests',
    brokers: config.kafkaBrokers,
    logLevel: logLevel.NOTHING,
  })
  producer = kafka.producer({ allowAutoTopicCreation: true })
  await producer.connect()
  return producer
}

export function newRunID(): string {
  return randomUUID()
}

/**
 * Ждет, пока backend-конsumer вступит в группу.
 * Нужно потому, что StartOffset=last: сообщения, опубликованные до join группы,
 * конsumer пропускает.
 */
export async function waitForConsumerGroup(
  groupID: string,
  timeoutMs = 60_000,
): Promise<void> {
  const kafka = new Kafka({
    clientId: 'e2e-tests-admin',
    brokers: config.kafkaBrokers,
    logLevel: logLevel.NOTHING,
  })
  const admin = kafka.admin()
  await admin.connect()
  const deadline = Date.now() + timeoutMs
  try {
    while (Date.now() < deadline) {
      try {
        const { groups } = await admin.listGroups()
        if (groups.some((x) => x.groupId === groupID)) {
          const described = await admin.describeGroups([groupID])
          if (described.groups[0]?.state === 'Stable') return
        }
      } catch {
        // брокер еще не готов — повторяем
      }
      await new Promise((r) => setTimeout(r, 1000))
    }
    throw new Error(`consumer group ${groupID} not stable after ${timeoutMs}ms`)
  } finally {
    await admin.disconnect()
  }
}

/** Демо-траектория: 5 шагов, включая error-шаг и успешный вызов инструмента. */
export function demoSteps(runID: string): StepMessage[] {
  const base = {
    run_id: runID,
    harness_name: 'e2e-harness',
    agent_version: 'e2e-1.0.0',
  }
  return [
    {
      ...base,
      step_number: 1,
      thought: 'Inspecting the repository structure.',
      status: 'success',
      latency_ms: 120,
      tokens_used: 300,
      cost_usd: 0.006,
    },
    {
      ...base,
      step_number: 2,
      thought: 'Listing files in the working directory.',
      tool_called: 'bash',
      status: 'success',
      latency_ms: 250,
      tokens_used: 450,
      cost_usd: 0.009,
    },
    {
      ...base,
      step_number: 3,
      thought: 'Reading the config loader.',
      tool_called: 'read_file',
      status: 'success',
      latency_ms: 180,
      tokens_used: 400,
      cost_usd: 0.008,
    },
    {
      ...base,
      step_number: 4,
      thought: 'Running tests, suite failed on a flaky fixture.',
      tool_called: 'bash',
      status: 'error',
      latency_ms: 900,
      tokens_used: 700,
      cost_usd: 0.014,
    },
    {
      ...base,
      step_number: 5,
      thought: 'Tests are green, summarizing the result.',
      status: 'success',
      latency_ms: 150,
      tokens_used: 350,
      cost_usd: 0.007,
    },
  ]
}

/** Публикует шаги в топик одним batch, key = run_id (порядок в одной партиции). */
export async function publishRun(steps: StepMessage[]): Promise<void> {
  const p = await getProducer()
  await p.send({
    topic: config.kafkaTopic,
    acks: -1,
    messages: steps.map((s) => ({
      key: s.run_id,
      value: JSON.stringify(s),
    })),
  })
}

export async function disconnectProducer(): Promise<void> {
  if (producer) {
    await producer.disconnect()
    producer = null
  }
}
