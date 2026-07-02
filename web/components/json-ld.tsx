/**
 * Renders a JSON-LD <script> for structured data (rich results in Google).
 * Server component — safe to embed the serialized data directly.
 */
export function JsonLd({ data }: { data: Record<string, unknown> | Record<string, unknown>[] }) {
  return (
    <script
      type="application/ld+json"
      // Structured data is trusted, statically authored content.
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  )
}
