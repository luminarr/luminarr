import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";
import { Shell } from "@/layouts/Shell";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import Dashboard from "@/pages/dashboard/Dashboard";
import MovieDetail from "@/pages/movies/MovieDetail";
import Queue from "@/pages/queue/Queue";
import SystemPage from "@/pages/settings/system/SystemPage";
import LibraryList from "@/pages/settings/libraries/LibraryList";
import QualityProfileList from "@/pages/settings/quality-profiles/QualityProfileList";
import IndexerList from "@/pages/settings/indexers/IndexerList";
import DownloadClientList from "@/pages/settings/download-clients/DownloadClientList";
import NotificationList from "@/pages/settings/notifications/NotificationList";
import ImportPage from "@/pages/settings/import/ImportPage";
import HistoryPage from "@/pages/history/HistoryPage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
    },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ErrorBoundary>
          <Routes>
            <Route element={<Shell />}>
              <Route index element={<ErrorBoundary><Dashboard /></ErrorBoundary>} />
              <Route path="movies/:id" element={<ErrorBoundary><MovieDetail /></ErrorBoundary>} />
              <Route path="queue" element={<ErrorBoundary><Queue /></ErrorBoundary>} />
              <Route path="history" element={<ErrorBoundary><HistoryPage /></ErrorBoundary>} />
              <Route path="settings">
                <Route path="libraries" element={<ErrorBoundary><LibraryList /></ErrorBoundary>} />
                <Route path="quality-profiles" element={<ErrorBoundary><QualityProfileList /></ErrorBoundary>} />
                <Route path="indexers" element={<ErrorBoundary><IndexerList /></ErrorBoundary>} />
                <Route path="download-clients" element={<ErrorBoundary><DownloadClientList /></ErrorBoundary>} />
                <Route path="notifications" element={<ErrorBoundary><NotificationList /></ErrorBoundary>} />
                <Route path="system" element={<ErrorBoundary><SystemPage /></ErrorBoundary>} />
                <Route path="import" element={<ErrorBoundary><ImportPage /></ErrorBoundary>} />
              </Route>
            </Route>
          </Routes>
        </ErrorBoundary>
        <Toaster
          position="bottom-right"
          toastOptions={{
            style: {
              background: "var(--color-bg-elevated)",
              border: "1px solid var(--color-border-default)",
              color: "var(--color-text-primary)",
              fontSize: 13,
            },
          }}
        />
      </BrowserRouter>
    </QueryClientProvider>
  );
}
