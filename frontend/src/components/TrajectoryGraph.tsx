import { Background, BackgroundVariant, Controls, MiniMap, ReactFlow } from '@xyflow/react'
import '@xyflow/react/dist/base.css'
import AgentStepNode from './AgentStepNode'
import type { AgentStepFlowNode, FlowEdge } from '../types'

const nodeTypes = { AgentStepNode }

interface TrajectoryGraphProps {
  nodes: AgentStepFlowNode[]
  edges: FlowEdge[]
}

export default function TrajectoryGraph({ nodes, edges }: TrajectoryGraphProps) {
  return (
    <div className="h-full w-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        minZoom={0.2}
        proOptions={{ hideAttribution: false }}
      >
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
        <Controls />
        <MiniMap pannable zoomable />
      </ReactFlow>
    </div>
  )
}
