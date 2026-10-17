import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')

function parseEnvFile(content: string): Record<string, string> {
  const out: Record<string, string> = {}
  for (const raw of content.split('\n')) {
    const line = raw.trim()
    if (!line || line.startsWith('#')) continue
    const eq = line.indexOf('=')
    if (eq < 0) continue
    const key = line.slice(0, eq).trim()
    let value = line.slice(eq + 1).trim()
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1)
    }
    out[key] = value
  }
  return out
}

function loadEnv(): Record<string, string> {
  let fileEnv: Record<string, string> = {}
  try {
    fileEnv = parseEnvFile(readFileSync(path.join(rootDir, '.env'), 'utf8'))
  } catch {
    // .env отсутствует — используем дефолты
  }
  const merged: Record<string, string> = { ...fileEnv }
  for (const [k, v] of Object.entries(process.env)) {
    if (v !== undefined && v !== '') merged[k] = v
  }
  return merged
}

const env = loadEnv()

export const rootEnv: Record<string, string> = { ...env }

export const config = {
  rootDir,
  kafkaBrokers: (env.KAFKA_BROKERS ?? 'localhost:9092').split(',').map((s) => s.trim()),
  kafkaTopic: env.KAFKA_TOPIC ?? 'agent-harness-metrics',
  kafkaGroupID: env.KAFKA_GROUP_ID ?? 'agent-metrics-analyzer',
  apiBasePath: env.API_BASE_PATH ?? '/v1',
  backendUrl: env.E2E_BACKEND_URL ?? `http://localhost${normalizeAddr(env.HTTP_ADDR ?? ':8080')}`,
  frontendUrl: env.E2E_FRONTEND_URL ?? `http://localhost:${env.FRONTEND_PORT ?? '5173'}`,
}

function normalizeAddr(addr: string): string {
  if (addr.startsWith(':')) return addr
  const idx = addr.indexOf(':')
  if (idx >= 0) return addr.slice(idx)
  return `:${addr}`
}
