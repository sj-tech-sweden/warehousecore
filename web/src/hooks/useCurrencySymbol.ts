import { useState, useEffect } from 'react';

let cachedSymbol: string | null = null;
let fetchPromise: Promise<string> | null = null;

async function fetchCurrencySymbol(): Promise<string> {
  if (cachedSymbol !== null) return cachedSymbol;
  if (fetchPromise) return fetchPromise;

  fetchPromise = fetch('/api/v1/config')
    .then(r => {
      if (!r.ok) {
        throw new Error(`Failed to fetch config: ${r.status} ${r.statusText}`);
      }
      return r.json();
    })
    .then((cfg: { currencySymbol?: string }) => {
      const symbol = cfg?.currencySymbol || '€';
      cachedSymbol = symbol;
      return symbol;
    })
    .catch(err => {
      const fallback = (window as any).__APP_CONFIG__?.currencySymbol || '€';
      console.error('[useCurrencySymbol] Failed to fetch config, using fallback:', err);
      cachedSymbol = fallback;
      return fallback;
    })
    .finally(() => {
      fetchPromise = null;
    });

  return fetchPromise;
}

export function useCurrencySymbol(): string {
  const [symbol, setSymbol] = useState<string>(
    cachedSymbol ?? (window as any).__APP_CONFIG__?.currencySymbol ?? '€',
  );

  useEffect(() => {
    fetchCurrencySymbol().then(s => setSymbol(s));

    const handleUpdate = (e: Event) => {
      const detail = (e as CustomEvent<{ symbol: string }>).detail;
      if (detail?.symbol) setSymbol(detail.symbol);
    };
    window.addEventListener('currency-symbol-updated', handleUpdate);
    return () => window.removeEventListener('currency-symbol-updated', handleUpdate);
  }, []);

  return symbol;
}

export function invalidateCurrencyCache() {
  cachedSymbol = null;
  fetchPromise = null;
}
