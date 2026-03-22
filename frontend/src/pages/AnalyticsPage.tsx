import { AnalyticsPanel } from '../components/AnalyticsPanel';
import type { Locale } from '../i18n';

type AnalyticsPageProps = {
  locale: Locale;
};

export function AnalyticsPage({ locale }: AnalyticsPageProps) {
  return (
    <section className="mt-3 px-2 sm:px-3">
      <AnalyticsPanel locale={locale} />
    </section>
  );
}
