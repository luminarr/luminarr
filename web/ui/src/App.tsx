import type { ReactNode } from "react";
import { BrowserRouter, Routes, Route, useLocation } from "react-router-dom";
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
import QualityDefinitionsPage from "@/pages/settings/quality-definitions/QualityDefinitionsPage";
import IndexerList from "@/pages/settings/indexers/IndexerList";
import DownloadClientList from "@/pages/settings/download-clients/DownloadClientList";
import NotificationList from "@/pages/settings/notifications/NotificationList";
import MediaServerList from "@/pages/settings/media-servers/MediaServerList";
import ImportListList from "@/pages/settings/import-lists/ImportListList";
import ImportExclusions from "@/pages/settings/import-lists/ImportExclusions";
import ImportPage from "@/pages/settings/import/ImportPage";
import BlocklistPage from "@/pages/settings/blocklist/BlocklistPage";
import MediaManagementPage from "@/pages/settings/media-management/MediaManagementPage";
import MediaScanningPage from "@/pages/settings/media-scanning/MediaScanningPage";
import AppSettingsPage from "@/pages/settings/app/AppSettingsPage";
import ActivityPage from "@/pages/activity/ActivityPage";
import HistoryPage from "@/pages/history/HistoryPage";
import WantedPage from "@/pages/wanted/WantedPage";
import CalendarPage from "@/pages/calendar/CalendarPage";
import StatsPage from "@/pages/stats/StatsPage";
import CollectionsPage from "@/pages/collections/CollectionsPage";
import CollectionDetail from "@/pages/collections/CollectionDetail";
import LibrarySyncPage from "@/pages/library-sync/LibrarySyncPage";
import CustomFormatsPage from "@/pages/settings/custom-formats/CustomFormatsPage";

function RouteEB({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  return <ErrorBoundary resetKey={pathname}>{children}</ErrorBoundary>;
}

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
              <Route index element={<RouteEB><Dashboard /></RouteEB>} />
              <Route path="activity" element={<RouteEB><ActivityPage /></RouteEB>} />
              <Route path="movies/:id" element={<RouteEB><MovieDetail /></RouteEB>} />
              <Route path="queue" element={<RouteEB><Queue /></RouteEB>} />
              <Route path="history" element={<RouteEB><HistoryPage /></RouteEB>} />
              <Route path="calendar" element={<RouteEB><CalendarPage /></RouteEB>} />
              <Route path="wanted" element={<RouteEB><WantedPage /></RouteEB>} />
              <Route path="stats" element={<RouteEB><StatsPage /></RouteEB>} />
              <Route path="collections" element={<RouteEB><CollectionsPage /></RouteEB>} />
              <Route path="collections/:id" element={<RouteEB><CollectionDetail /></RouteEB>} />
              <Route path="library-sync" element={<RouteEB><LibrarySyncPage /></RouteEB>} />
              <Route path="settings">
                <Route path="app" element={<RouteEB><AppSettingsPage /></RouteEB>} />
                <Route path="libraries" element={<RouteEB><LibraryList /></RouteEB>} />
                <Route path="quality-profiles" element={<RouteEB><QualityProfileList /></RouteEB>} />
                <Route path="quality-definitions" element={<RouteEB><QualityDefinitionsPage /></RouteEB>} />
                <Route path="custom-formats" element={<RouteEB><CustomFormatsPage /></RouteEB>} />
                <Route path="indexers" element={<RouteEB><IndexerList /></RouteEB>} />
                <Route path="download-clients" element={<RouteEB><DownloadClientList /></RouteEB>} />
                <Route path="notifications" element={<RouteEB><NotificationList /></RouteEB>} />
                <Route path="media-servers" element={<RouteEB><MediaServerList /></RouteEB>} />
                <Route path="import-lists" element={<RouteEB><ImportListList /></RouteEB>} />
                <Route path="import-exclusions" element={<RouteEB><ImportExclusions /></RouteEB>} />
                <Route path="blocklist" element={<RouteEB><BlocklistPage /></RouteEB>} />
                <Route path="media-management" element={<RouteEB><MediaManagementPage /></RouteEB>} />
                <Route path="media-scanning" element={<RouteEB><MediaScanningPage /></RouteEB>} />
                <Route path="system" element={<RouteEB><SystemPage /></RouteEB>} />
                <Route path="import" element={<RouteEB><ImportPage /></RouteEB>} />
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
