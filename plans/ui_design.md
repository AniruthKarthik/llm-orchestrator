# Plan: Comprehensive Web UI for LLM Orchestrator

## Background & Motivation
The LLM Orchestrator currently lacks a graphical interface, making it difficult for users to manage complex workflows, visualize task dependencies, and monitor executions in real-time. This plan introduces a professional web UI to provide a centralized management console.

## Scope & Impact
- **Impact:** Significant improvement in user experience, easier workflow management, and better observability.
- **Scope:** 
    - Full-featured dashboard (React/Next.js).
    - Workflow Builder (Drag-and-drop).
    - Real-time execution monitoring.
    - Configuration management (.env and docker-compose).
    - API enhancements to support the UI.

## Proposed Solution

### Frontend Architecture (Next.js + Ant Design)
- **Framework:** Next.js (App Router) for SEO, performance, and easy API routing if needed.
- **UI Components:** Ant Design (AntD) for a professional, data-rich dashboard aesthetic.
- **State Management:** TanStack Query (React Query) for efficient data fetching and caching.
- **Workflow Visualization:** React Flow for the interactive DAG (Directed Acyclic Graph) builder.
- **Real-time Updates:** WebSockets for live logs and task status changes.

### Backend Enhancements (Go)
- **API Additions:**
    - CRUD endpoints for Workflows, Tasks, and Agents.
    - Metadata endpoints: `/api/providers`, `/api/models`, `/api/agents`.
    - System endpoints: `/api/config` (read/write .env and docker-compose.yaml).
    - Real-time stream: `/api/ws` for events.
- **Event Integration:** Hook into the existing `internal/events/bus.go` to broadcast events to connected clients.

### UI Features
1.  **Dashboard Home:** High-level metrics (success rate, active runs, cost estimation).
2.  **Workflow Studio:**
    - Node-based editor (React Flow) to define task dependencies.
    - Sidebar with "Task Palettes" (different LLM types, tool-use, human-in-the-loop).
    - Property pane for task configuration (prompts, schemas, retries).
3.  **Execution Tracker:**
    - Live DAG view highlighting currently running nodes.
    - Gantt chart for execution timeline.
    - Side-by-side view of Task Input/Output/Logs.
4.  **Settings:**
    - API Key management (edits .env).
    - Docker control (edits docker-compose.yaml).

## Implementation Plan

### Phase 1: Foundation (Turns 1-5)
1. Initialize Next.js project in `ui/` directory.
2. Configure Ant Design, theme, and basic layout (Sidebar, Header).
3. Set up React Query and basic API client.
4. Expand Go backend with `GET /api/v1/meta/providers` and `GET /api/v1/workflows` endpoints.

### Phase 2: Workflow Studio (Turns 6-15)
1. Integrate React Flow.
2. Build custom nodes for Orchestrator tasks.
3. Implement the property editor for tasks (dynamic forms using AntD `Form`).
4. Implement persistence: saving the DAG to the Go backend.

### Phase 3: Execution & Real-time (Turns 16-25)
1. Implement the Execution Detail view.
2. Add WebSocket support to the Go backend (using `gorilla/websocket`).
3. Connect the UI to the WebSocket for real-time task status updates.
4. Build the Log Viewer component.

### Phase 4: System Management (Turns 26-30)
1. Implement the Configuration editor (.env/docker-compose).
2. Add "Run Now" triggers and execution history.
3. Final polish and responsive design tweaks.

## Verification Plan
- **Unit Tests:** For critical UI logic and API transformations.
- **Integration Tests:** End-to-end flow from creating a workflow in the UI to seeing it execute in real-time.
- **Manual QA:** Verify that editing `.env` correctly updates the backend's environment.

## Alternatives Considered
- **Vite SPA:** Good, but Next.js offers more out-of-the-box for "Backend for Frontend" needs if we want to proxy certain local system calls.
- **Tailwind CSS:** Highly customizable, but Ant Design is faster for building "professional app dashboards" with complex data components.
- **D3.js for Graphs:** More powerful but React Flow is much easier to integrate with React state for interactive builders.
