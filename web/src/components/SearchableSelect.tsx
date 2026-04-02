import { useEffect, useId, useRef, useState } from 'react';
import { ChevronDown, Search, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export interface SearchableSelectOption {
  value: string;
  label: string;
}

export interface SearchableSelectProps {
  value: string;
  onChange: (value: string) => void;
  options: SearchableSelectOption[];
  placeholder?: string;
  disabled?: boolean;
  required?: boolean;
  /** Applied to the outer container (for sizing/layout) and the interactive control (button/input). */
  className?: string;
  title?: string;
}

export function SearchableSelect({
  value,
  onChange,
  options,
  placeholder,
  disabled = false,
  required = false,
  className = '',
  title,
}: SearchableSelectProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [activeIndex, setActiveIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listboxId = useId();
  const optionIdPrefix = useId();

  const selectedOption = options.find(opt => opt.value === value);

  const filteredOptions = searchText
    ? options.filter(opt => opt.label.toLowerCase().includes(searchText.toLowerCase()))
    : options;

  useEffect(() => {
    if (!isOpen) return;
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setSearchText('');
        setActiveIndex(-1);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  useEffect(() => {
    if (isOpen) {
      inputRef.current?.focus();
      setActiveIndex(-1);
    }
  }, [isOpen]);

  // Reset keyboard cursor when filtered list changes
  useEffect(() => {
    setActiveIndex(-1);
  }, [searchText]);

  const handleOpen = () => {
    if (disabled) return;
    setIsOpen(true);
    setSearchText('');
  };

  const handleSelect = (optionValue: string) => {
    onChange(optionValue);
    setIsOpen(false);
    setSearchText('');
    setActiveIndex(-1);
  };

  const handleClear = () => {
    onChange('');
    setIsOpen(false);
    setSearchText('');
    setActiveIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'Escape':
        setIsOpen(false);
        setSearchText('');
        setActiveIndex(-1);
        break;
      case 'ArrowDown':
        e.preventDefault();
        if (filteredOptions.length > 0) {
          setActiveIndex(prev => (prev < filteredOptions.length - 1 ? prev + 1 : 0));
        }
        break;
      case 'ArrowUp':
        e.preventDefault();
        if (filteredOptions.length > 0) {
          setActiveIndex(prev => (prev > 0 ? prev - 1 : filteredOptions.length - 1));
        }
        break;
      case 'Enter':
        e.preventDefault();
        if (activeIndex >= 0 && activeIndex < filteredOptions.length) {
          handleSelect(filteredOptions[activeIndex].value);
        } else if (filteredOptions.length === 1) {
          handleSelect(filteredOptions[0].value);
        }
        break;
    }
  };

  return (
    <div ref={containerRef} className={`relative ${className}`} title={title}>
      {/* Hidden native select for form validation */}
      <select
        value={value}
        onChange={e => onChange(e.target.value)}
        required={required}
        disabled={disabled}
        aria-hidden="true"
        tabIndex={-1}
        className="sr-only"
      >
        {!options.some(opt => opt.value === '') && (
          <option value="">{placeholder ?? ''}</option>
        )}
        {options.map(opt => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>

      {isOpen ? (
        <div>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" aria-hidden="true" />
            <input
              ref={inputRef}
              role="combobox"
              aria-expanded={true}
              aria-controls={listboxId}
              aria-haspopup="listbox"
              aria-autocomplete="list"
              aria-activedescendant={activeIndex >= 0 && activeIndex < filteredOptions.length ? `${optionIdPrefix}-${activeIndex}` : undefined}
              type="text"
              value={searchText}
              onChange={e => setSearchText(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={t('common.search')}
              className={`w-full rounded-lg border border-accent-red bg-white/10 py-2 pl-9 pr-3 text-white placeholder-gray-500 outline-none ${className}`}
            />
          </div>
          <div
            id={listboxId}
            role="listbox"
            className="absolute z-50 mt-1 max-h-60 w-full overflow-y-auto rounded-lg border border-white/20 bg-gray-900 shadow-xl"
          >
            {filteredOptions.length === 0 ? (
              <div className="px-3 py-2 text-sm text-gray-500">{t('common.noResults')}</div>
            ) : (
              filteredOptions.map((opt, index) => (
                <div
                  key={opt.value}
                  id={`${optionIdPrefix}-${index}`}
                  role="option"
                  aria-selected={opt.value === value}
                  onClick={() => handleSelect(opt.value)}
                  className={`cursor-pointer px-3 py-2 text-sm hover:bg-white/10 ${
                    index === activeIndex
                      ? 'bg-white/20 text-white'
                      : opt.value === value
                        ? 'bg-white/15 text-accent-red'
                        : 'text-white'
                  }`}
                >
                  {opt.label}
                </div>
              ))
            )}
          </div>
        </div>
      ) : (
        <button
          type="button"
          role="combobox"
          aria-expanded={false}
          aria-controls={listboxId}
          aria-haspopup="listbox"
          onClick={handleOpen}
          disabled={disabled}
          className={`flex w-full items-center justify-between rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-left text-sm transition focus:outline-none focus:ring-1 focus:ring-accent-red disabled:cursor-not-allowed disabled:opacity-50 ${
            selectedOption ? 'text-white' : 'text-gray-400'
          } ${className}`}
        >
          <span className="truncate">{selectedOption ? selectedOption.label : (placeholder ?? t('common.select'))}</span>
          <span className="ml-2 flex shrink-0 items-center gap-1">
            {value && !disabled && (
              <span
                role="button"
                tabIndex={0}
                onClick={event => {
                  event.stopPropagation();
                  handleClear();
                }}
                onKeyDown={event => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    event.stopPropagation();
                    handleClear();
                  }
                }}
                className="p-0 text-gray-400 hover:text-white"
                aria-label={t('common.clear')}
              >
                <X className="h-3.5 w-3.5" aria-hidden="true" />
              </span>
            )}
            <ChevronDown className="h-4 w-4 text-gray-400" aria-hidden="true" />
          </span>
        </button>
      )}
    </div>
  );
}
