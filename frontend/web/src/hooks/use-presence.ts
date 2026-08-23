import { useEffect, useState } from "react";
import { useRealtime } from "../components/RealtimeProvider";

export function usePresence(scope: string) {
  const [count, setCount] = useState(0);
  const { connected, subscribePresence, send } = useRealtime();
  useEffect(() => {
    if (!scope || !connected) return;
    const unsubscribe = subscribePresence((eventScope, nextCount) => {
      if (eventScope === scope) setCount(nextCount);
    });
    send({ op: "presence.join", scope });
    return () => {
      unsubscribe();
      send({ op: "presence.leave", scope });
      setCount(0);
    };
  }, [connected, scope, send, subscribePresence]);
  return count;
}
