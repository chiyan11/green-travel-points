import type { ReactNode } from 'react'

interface PlaceholderPageProps {
  title: string
  sprint: string
  summary: string
  tasks: readonly string[]
}

/** Sprint 0 占位页：明确"这一页由谁、在哪个 Sprint 交付"，避免分工真空。 */
export function PlaceholderPage({ title, sprint, summary, tasks }: PlaceholderPageProps): ReactNode {
  return (
    <>
      <section className="hero">
        <h2>{title}</h2>
        <p>{summary}</p>
      </section>

      <section className="card">
        <h3>
          交付计划<span className="badge badge--muted">{sprint}</span>
        </h3>
        <ul className="checklist">
          {tasks.map((task) => (
            <li key={task}>{task}</li>
          ))}
        </ul>
      </section>
    </>
  )
}
