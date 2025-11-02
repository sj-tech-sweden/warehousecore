import { Languages } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export function LanguageSwitcher() {
  const { i18n } = useTranslation();

  const languages = [
    { code: 'de', name: 'Deutsch', flag: '🇩🇪' },
    { code: 'en', name: 'English', flag: '🇺🇸' },
  ];

  const changeLanguage = (langCode: string) => {
    i18n.changeLanguage(langCode);
    localStorage.setItem('warehousecore_language', langCode);
  };

  return (
    <div className="relative group">
      <button
        className="flex items-center gap-2 px-3 py-2 rounded-lg bg-white/5 hover:bg-white/10 transition-colors"
        title="Change Language"
      >
        <Languages className="w-5 h-5 text-gray-400" />
        <span className="text-sm text-gray-400 hidden sm:inline">
          {languages.find((l) => l.code === i18n.language)?.flag || '🌐'}
        </span>
      </button>

      {/* Dropdown */}
      <div className="absolute right-0 mt-2 w-48 bg-dark-card border border-white/10 rounded-lg shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200 z-50">
        {languages.map((lang) => (
          <button
            key={lang.code}
            onClick={() => changeLanguage(lang.code)}
            className={`w-full flex items-center gap-3 px-4 py-3 text-left transition-colors ${
              i18n.language === lang.code
                ? 'bg-accent-red/20 text-accent-red'
                : 'text-gray-300 hover:bg-white/5'
            }`}
          >
            <span className="text-xl">{lang.flag}</span>
            <span className="font-medium">{lang.name}</span>
            {i18n.language === lang.code && (
              <span className="ml-auto text-accent-red">✓</span>
            )}
          </button>
        ))}
      </div>
    </div>
  );
}
