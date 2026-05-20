# LLM Orchestrator: UI

The frontend for LLM Orchestrator is a modern, reactive web application designed for visualizing, building, and monitoring AI workflows.

---

## Tech Stack

- **Framework:** [React](https://reactjs.org/) (TypeScript)
- **Build Tool:** [Vite](https://vitejs.dev/)
- **State Management:** [Zustand](https://github.com/pmndrs/zustand)
- **Workflow Visualization:** [React Flow](https://reactflow.dev/)
- **Styling:** Vanilla CSS (for maximum control and performance)
- **Icons:** Custom SVG sprites

---

## Project Structure

- `src/api`: Axios client and API service definitions.
- `src/components/builder`: Components specific to the Workflow Builder (e.g., custom Task nodes).
- `src/components/ui`: Reusable UI primitives (Buttons, Modals, Toasts).
- `src/context`: React Context providers (e.g., for WebSocket management).
- `src/hooks`: Custom React hooks for data fetching, theme management, and WebSocket events.
- `src/pages`: Top-level page components (Builder, Dashboard, Executions).
- `src/store`: Zustand stores for global state (Workflows, Config, Notifications).

---

## Development

### Prerequisites
- **Node.js:** 18+
- **npm:** 9+

### Setup
1.  **Install Dependencies:**
    ```bash
    npm install
    ```

2.  **Start Development Server:**
    ```bash
    npm run dev
    ```
    The UI will be available at `http://localhost:5173`. By default, it expects the backend API to be running at `http://localhost:8080`.

### Production Build
```bash
npm run build
```
The build artifacts will be located in the `dist/` directory, which can be served by the Go backend or any static web server.

---

## Real-time Updates

The UI integrates with the backend via WebSockets (`/api/v1/ws`). This allows the application to:
- Receive live updates on task status (e.g., transition from `PENDING` to `RUNNING` to `COMPLETED`).
- Stream logs and execution events directly to the dashboard.
- Update token usage metrics in real-time.

---

## Design Principles

- **Clarity:** Workflows should be easy to read and navigate, even as they grow in complexity.
- **Responsiveness:** Immediate visual feedback for all user actions (node dragging, execution triggers).
- **Aesthetics:** A modern, clean look that emphasizes the structured nature of DAGs.
