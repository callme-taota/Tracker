import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, type FeatureFlagSnapshot } from '@/api'

type FeatureFlagContextValue = {
  snapshot: FeatureFlagSnapshot | null
  loading: boolean
  variant: (key: string) => string
}

const FeatureFlagContext = createContext<FeatureFlagContextValue>({
  snapshot: null,
  loading: true,
  variant: () => '',
})

export function FeatureFlagProvider({ children }: { children: ReactNode }) {
  const [snapshot, setSnapshot] = useState<FeatureFlagSnapshot | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    api
      .getFeatureFlagSnapshot()
      .then((next) => {
        if (active) setSnapshot(next)
      })
      .catch(() => {
        if (active) setSnapshot(null)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  const value = useMemo<FeatureFlagContextValue>(
    () => ({
      snapshot,
      loading,
      variant: (key: string) => snapshot?.flags?.[key]?.variant ?? '',
    }),
    [loading, snapshot],
  )

  return <FeatureFlagContext.Provider value={value}>{children}</FeatureFlagContext.Provider>
}

export function useFeatureFlags() {
  return useContext(FeatureFlagContext)
}
