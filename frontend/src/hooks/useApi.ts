import { useCallback, useEffect, useRef, useState } from 'react';

export function useApi<T>(load: () => Promise<T>, deps: unknown[] = []) {
  const [data, setData] = useState<T>();
  const [error, setError] = useState<Error>();
  const [loading, setLoading] = useState(true);

  const loadRef = useRef(load);
  useEffect(() => {
    loadRef.current = load;
  }, [load]);

  const reload = useCallback(() => {
    setLoading(true);
    setError(undefined);
    return loadRef.current()
      .then(setData)
      .catch((e: unknown) => setError(e instanceof Error ? e : new Error('Unknown error')))
      .finally(() => setLoading(false));
  }, []);

  const depKey = JSON.stringify(deps);
  useEffect(() => {
    let mounted = true;
    const executeReload = async () => {
      setLoading(true);
      setError(undefined);
      try {
        const result = await loadRef.current();
        if (mounted) setData(result);
      } catch (e) {
        if (mounted) setError(e instanceof Error ? e : new Error('Unknown error'));
      } finally {
        if (mounted) setLoading(false);
      }
    };
    executeReload();
    return () => {
      mounted = false;
    };
    }, [depKey]);

  return { data, error, loading, reload };
}