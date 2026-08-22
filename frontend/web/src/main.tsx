import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";
import { AetherProvider, AetherProviderProps } from "@aether/design-system";
import { NotificationProvider } from "./components/NotificationProvider";
import { RealtimeProvider } from "./components/RealtimeProvider";
import { OrgProvider } from "./components/OrgProvider";
import "@aether/design-system/styles.css";
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
const config: AetherProviderProps['config']= {
   
};
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

ReactDOM.createRoot(document.getElementById("root")!).render(
    <QueryClientProvider client={queryClient}>
      <AetherProvider defaultTheme="dark" persist storageKey="aether-theme" position="bottom-right">
        <RealtimeProvider>
          <NotificationProvider>
            <OrgProvider>
              <RouterProvider router={router} />
            </OrgProvider>
          </NotificationProvider>
        </RealtimeProvider>
      </AetherProvider>
    </QueryClientProvider>
);
