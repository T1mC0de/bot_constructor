import { useState } from 'react'
import OrdersList from './OrdersList.tsx'
import SurveyEditor from './SurveyEditor.tsx'

const tabs = [
  { id: 'survey', label: 'Анкета' },
  { id: 'orders', label: 'Заявки' },
] as const

type Tab = (typeof tabs)[number]['id']

export default function App() {
  const [tab, setTab] = useState<Tab>('survey')

  return (
    <div className="app">
      <header className="app-header">
        <h1>Конструктор ботов</h1>
        <nav className="tabs" role="tablist" aria-label="Разделы">
          {tabs.map((t) => (
            <button
              key={t.id}
              id={`tab-${t.id}`}
              type="button"
              role="tab"
              aria-selected={tab === t.id}
              aria-controls={`panel-${t.id}`}
              onClick={() => setTab(t.id)}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </header>

      {/* The editor stays mounted so unsaved changes survive a look at the
          orders; the orders are loaded again each time their tab is opened. */}
      <main id="panel-survey" role="tabpanel" aria-labelledby="tab-survey" hidden={tab !== 'survey'}>
        <SurveyEditor />
      </main>
      {tab === 'orders' && (
        <main id="panel-orders" role="tabpanel" aria-labelledby="tab-orders">
          <OrdersList />
        </main>
      )}
    </div>
  )
}
