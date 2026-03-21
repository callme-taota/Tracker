import { Link, Outlet, useLocation, matchPath } from 'react-router-dom'
import { ChevronDown, LayoutDashboard, Network, Plug } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { useState } from 'react'

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
  const [dataOpen, setDataOpen] = useState(true)
  const isEditor = !!matchPath('/pipelines/:id', loc.pathname)

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-40 flex h-14 shrink-0 items-center gap-4 border-b bg-card px-4">
        <Link to="/pipelines" className="text-lg font-semibold tracking-tight">
          Tracker
        </Link>
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
          <div className="relative">
            <Button
              variant="ghost"
              size="sm"
              className="gap-1"
              onClick={() => setDataOpen((o) => !o)}
              aria-expanded={dataOpen}
            >
              数据
              <ChevronDown className={cn('h-4 w-4 transition-transform', dataOpen && 'rotate-180')} />
            </Button>
            {dataOpen ? (
              <div className="absolute left-0 top-full z-50 mt-1 min-w-[10rem] rounded-md border bg-popover p-1 shadow-md">
                {dataNav.map(({ to, label }) => (
                  <Button key={to} variant="ghost" size="sm" className="w-full justify-start font-normal" asChild>
                    <Link to={to}>{label}</Link>
                  </Button>
                ))}
              </div>
            ) : null}
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
