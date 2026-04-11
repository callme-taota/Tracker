import { Link, Outlet, useLocation, matchPath } from 'react-router-dom'
import { ChevronDown, LayoutDashboard, Network, Plug } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { useFeatureFlags } from '@/lib/feature-flags'

const primaryNav = [{ to: '/pipelines', label: '管道', icon: Network }]

const dataNav = [
  { to: '/sources', label: '数据源' },
  { to: '/items', label: '抓取条目' },
  { to: '/summaries', label: '摘要' },
  { to: '/interests', label: '兴趣' },
]

const toolsNav = [
  { to: '/', label: '概览', icon: LayoutDashboard },
  { to: '/pipeline', label: '运行状态' },
  { to: '/plugins', label: '插件', icon: Plug },
]

export function AppShell() {
  const loc = useLocation()
  const isEditor = !!matchPath('/pipelines/:id', loc.pathname)
  const { snapshot, variant } = useFeatureFlags()
  const showRuntimeExperiment = variant('web.runtime_experiments') === 'on'

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-40 flex h-14 shrink-0 items-center gap-4 border-b bg-card px-4">
        <Link to="/pipelines" className="text-lg font-semibold tracking-tight">
          Tracker
        </Link>
        {showRuntimeExperiment ? (
          <Badge variant="secondary" className="hidden sm:inline-flex">
            实验中 {snapshot?.release.channel ?? 'stable'}/{snapshot?.release.ring ?? 'global'}
          </Badge>
        ) : null}
        <Separator orientation="vertical" className="h-6" />
        <nav className="flex flex-1 flex-wrap items-center gap-1">
          {primaryNav.map(({ to, label, icon: Icon }) => (
            <Button key={to} variant={loc.pathname.startsWith(to) ? 'secondary' : 'ghost'} size="sm" asChild>
              <Link to={to} className="gap-2">
                <Icon className="h-4 w-4" />
                {label}
              </Link>
            </Button>
          ))}
          {/* 默认收起；悬浮展开。pt-1 作为热区桥接，避免从按钮移到菜单时闪关 */}
          <div className="group relative">
            <Button variant="ghost" size="sm" className="gap-1" type="button" aria-haspopup="true">
              数据
              <ChevronDown
                className={cn(
                  'h-4 w-4 transition-transform duration-150',
                  'group-hover:rotate-180 group-focus-within:rotate-180',
                )}
              />
            </Button>
            <div
              className={cn(
                'pointer-events-none invisible absolute left-0 top-full z-50 pt-1 opacity-0 transition-opacity duration-150',
                'group-hover:pointer-events-auto group-hover:visible group-hover:opacity-100',
                'group-focus-within:pointer-events-auto group-focus-within:visible group-focus-within:opacity-100',
              )}
            >
              <div className="min-w-[10rem] rounded-md border bg-popover p-1 shadow-md" role="menu">
                {dataNav.map(({ to, label }) => (
                  <Button key={to} variant="ghost" size="sm" className="w-full justify-start font-normal" asChild>
                    <Link to={to} role="menuitem">
                      {label}
                    </Link>
                  </Button>
                ))}
              </div>
            </div>
          </div>
          <Separator orientation="vertical" className="mx-1 hidden h-6 sm:block" />
          {toolsNav.map((item) => {
            const Icon = item.icon
            const active =
              item.to === '/plugins'
                ? loc.pathname === '/plugins' || loc.pathname.startsWith('/plugins/')
                : loc.pathname === item.to
            return (
              <Button key={item.to} variant={active ? 'secondary' : 'ghost'} size="sm" asChild>
                <Link to={item.to} className={cn('gap-2', !Icon && 'pl-3')}>
                  {Icon ? <Icon className="h-4 w-4" /> : null}
                  {item.label}
                </Link>
              </Button>
            )
          })}
        </nav>
      </header>
      <main
        className={cn(
          'flex-1',
          isEditor
            ? 'flex min-h-0 flex-col overflow-hidden p-0 h-[calc(100vh-3.5rem)] max-h-[calc(100vh-3.5rem)]'
            : 'overflow-auto p-6',
        )}
      >
        <Outlet />
      </main>
    </div>
  )
}
