import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";
import { ToastProvider } from "./components/ui";
import { OverlayProvider } from "./components/OverlayManager";
import { CommandPaletteProvider } from "./components/command-palette";
import { NotificationProvider } from "./components/NotificationProvider";
import { RealtimeProvider } from "./components/RealtimeProvider";
import { OrgProvider } from "./components/OrgProvider";
import "./styles.css";
import "./runtime.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 2000,
      refetchOnWindowFocus: false,
    },
  },
});

const router = createRouter({ routeTree, defaultPreload: false, context: { queryClient } });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <CommandPaletteProvider>
        <OverlayProvider>
        <ToastProvider>
          <RealtimeProvider>
            <NotificationProvider>
              <OrgProvider>
                <RouterProvider router={router} />
              </OrgProvider>
            </NotificationProvider>
          </RealtimeProvider>
        </ToastProvider>
        </OverlayProvider>
      </CommandPaletteProvider>
    </QueryClientProvider>
  </React.StrictMode>
);
