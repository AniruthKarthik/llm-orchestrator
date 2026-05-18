import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { DashboardLayout } from '@/components/layout/DashboardLayout';
import { ErrorBoundary } from '@/components/ui/ErrorBoundary';
import { Toaster } from '@/components/ui/Toaster';
import DashboardPage from '@/pages/DashboardPage';
import WorkflowsPage from '@/pages/WorkflowsPage';
import WorkflowBuilderPage from '@/pages/WorkflowBuilderPage';
import ExecutionsPage from '@/pages/ExecutionsPage';
import ConfigPage from '@/pages/ConfigPage';
import ProvidersPage from '@/pages/ProvidersPage';
import AgentsPage from '@/pages/AgentsPage';
import ArtifactsPage from '@/pages/ArtifactsPage';
import QueuesPage from '@/pages/QueuesPage';

function App() {
  return (
    <Router>
      <ErrorBoundary fallbackTitle="Application Error">
        <DashboardLayout>
          <ErrorBoundary fallbackTitle="Page Error">
            <Routes>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/workflows" element={<WorkflowsPage />} />
              <Route path="/workflows/new" element={<WorkflowBuilderPage />} />
              <Route path="/workflows/:id" element={<WorkflowBuilderPage />} />
              <Route path="/executions" element={<ExecutionsPage />} />
              <Route path="/queues" element={<QueuesPage />} />
              <Route path="/agents" element={<AgentsPage />} />
              <Route path="/providers" element={<ProvidersPage />} />
              <Route path="/artifacts" element={<ArtifactsPage />} />
              <Route path="/config" element={<ConfigPage />} />
            </Routes>
          </ErrorBoundary>
        </DashboardLayout>
        {/* Global toast notification portal */}
        <Toaster />
      </ErrorBoundary>
    </Router>
  );
}

export default App;
