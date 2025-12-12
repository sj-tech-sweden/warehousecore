import { useEffect } from 'react';

/**
 * Hook to block body scroll when a modal is open
 * Prevents background page scrolling while modal is displayed
 */
export function useBlockBodyScroll(isOpen: boolean) {
  useEffect(() => {
    if (isOpen) {
      // Save current scroll position
      const scrollY = window.scrollY;

      // Block scroll on body
      document.body.style.overflow = 'hidden';
      document.body.style.position = 'fixed';
      document.body.style.top = `-${scrollY}px`;
      document.body.style.width = '100%';

      return () => {
        // Restore scroll
        document.body.style.overflow = '';
        document.body.style.position = '';
        document.body.style.top = '';
        document.body.style.width = '';
        window.scrollTo(0, scrollY);
      };
    }
  }, [isOpen]);
}
