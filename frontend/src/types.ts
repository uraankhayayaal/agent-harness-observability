import type { Node } from '@xyflow/react'

export type StepData = {
  stepNumber: number
  thought: string
  toolCalled: string
  status: 'success' | 'error'
  latencyMs: number
  tokensUsed: number
}

export type AgentStepFlowNode = Node<StepData, 'AgentStepNode'>

export interface FlowEdge {
  id: string
  source: string
  target: string
  animated?: boolean
}

export interface Trajectory {
  nodes: AgentStepFlowNode[]
  edges: FlowEdge[]
}
