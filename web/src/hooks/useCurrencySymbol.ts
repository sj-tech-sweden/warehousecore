import { useState, useEffect } from 'react';

let cachedSymbol: string | null = null;
let fetchPromise: Promise<string> | null = null;

async function fetchCurrencySymbol(): Promise<string> {
  if (cachedSymbol !== null) return cachedSymbol;
  if (fetchPromise) return fetchPromise;

  fetchPromise = fetch('/api/v1/config')
    .then(r => r.json())
    .then((cfg: { currencySymbol?: string }) => {
      const symbol = cfg?.currencySymbol || '€';
      cachedSymbol = symbol;
      return symbol;
    })
    .catch(err => {
      console.error('[useCurrencySymbol] Failed to fetch config, using fallback:', err);
      fetchPromise = null;
      return (window as any).__APP_CONFIG__?.currencySymbol || '€';
    });

  return fetchPromise;
}

export function useCurrencySymbol(): string {
  const [symbol, setSymbol] = useState<string>(
    cachedSymbol ?? (window as any).__APP_CONFIG__?.currencySymbol ?? '€',
  );

  useEffect(() => {
    fetchCurrencySymbol().then(s => setSymbol(s));
  }, []);

  return symbol;
}

export function invalidateCurrencyCache() {
  cachedSymbol = null;
  fetchPromise = null;
}
