import { useState, useEffect } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Film,
  Download,
  History,
  Library,
  Settings2,
  SlidersHorizontal,
  Gauge,
  Search,
  Bell,
  MonitorPlay,
  Server,
  ArrowDownToLine,
  Ban,
  Bookmark,
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  Activity,
  BarChart2,
  RefreshCw,
  Menu,
  X,
  ScanLine,
  Paintbrush,
  ListPlus,
  ShieldOff,
  Layers,
} from "lucide-react";
import { useSystemHealth } from "@/api/system";
import { useWebSocket } from "@/api/websocket";
import { applyTheme } from "@/theme";
import { useCommandPalette } from "@/components/command-palette/useCommandPalette";
import { CommandPalette } from "@/components/command-palette/CommandPalette";

interface NavItem {
  to: string;
  icon: React.ElementType;
  label: string;
}

const mainNav: NavItem[] = [
  { to: "/",          icon: LayoutDashboard, label: "Dashboard" },
  { to: "/activity",     icon: Activity,     label: "Activity" },
  { to: "/calendar",     icon: CalendarDays, label: "Calendar" },
  { to: "/wanted",       icon: Bookmark,     label: "Wanted" },
  { to: "/library-sync", icon: RefreshCw,    label: "Library Sync" },
  { to: "/stats",        icon: BarChart2,    label: "Statistics" },
  { to: "/queue",     icon: Download,        label: "Queue" },
  { to: "/history",   icon: History,         label: "History" },
];

const settingsNav: NavItem[] = [
  { to: "/settings/libraries",         icon: Library,          label: "Libraries" },
  { to: "/settings/media-management",  icon: Film,             label: "Media Management" },
  { to: "/settings/media-scanning",    icon: ScanLine,         label: "Media Scanning" },
  { to: "/settings/quality-profiles",   icon: SlidersHorizontal, label: "Quality Profiles" },
  { to: "/settings/quality-definitions", icon: Gauge,           label: "Quality Definitions" },
  { to: "/settings/custom-formats",    icon: Layers,          label: "Custom Formats" },
  { to: "/settings/indexers",          icon: Search,           label: "Indexers" },
  { to: "/settings/download-clients",  icon: Settings2,        label: "Download Clients" },
  { to: "/settings/notifications",     icon: Bell,             label: "Notifications" },
  { to: "/settings/media-servers",    icon: MonitorPlay,      label: "Media Servers" },
  { to: "/settings/import-lists",    icon: ListPlus,         label: "Import Lists" },
  { to: "/settings/import-exclusions", icon: ShieldOff,      label: "Import Exclusions" },
  { to: "/settings/blocklist",         icon: Ban,              label: "Blocklist" },
  { to: "/settings/import",            icon: ArrowDownToLine,  label: "Import" },
  { to: "/settings/system",            icon: Server,           label: "System" },
  { to: "/settings/app",               icon: Paintbrush,       label: "App Settings" },
];

function useIsMobile() {
  const [isMobile, setIsMobile] = useState(() => window.innerWidth < 768);
  useEffect(() => {
    const mq = window.matchMedia("(max-width: 767px)");
    const handler = (e: MediaQueryListEvent) => setIsMobile(e.matches);
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, []);
  return isMobile;
}

function SidebarNavItem({
  item,
  collapsed,
  onClick,
}: {
  item: NavItem;
  collapsed: boolean;
  onClick?: () => void;
}) {
  const Icon = item.icon;
  return (
    <NavLink
      to={item.to}
      end={item.to === "/"}
      title={collapsed ? item.label : undefined}
      onClick={onClick}
      style={({ isActive }) => ({
        display: "flex",
        alignItems: "center",
        gap: "10px",
        padding: "0 12px",
        height: "40px",
        borderRadius: "6px",
        textDecoration: "none",
        fontSize: "14px",
        fontWeight: 500,
        whiteSpace: "nowrap",
        overflow: "hidden",
        transition: "background 150ms ease, color 150ms ease",
        borderLeft: isActive ? "2px solid var(--color-accent)" : "2px solid transparent",
        background: isActive ? "var(--color-accent-muted)" : "transparent",
        color: isActive ? "var(--color-accent-hover)" : "var(--color-text-secondary)",
        marginLeft: "-2px",
      })}
    >
      <Icon size={18} strokeWidth={1.5} style={{ flexShrink: 0 }} />
      {!collapsed && <span>{item.label}</span>}
    </NavLink>
  );
}

function HealthDot({ collapsed }: { collapsed: boolean }) {
  const { data: health } = useSystemHealth();
  const allOk = !health || health.status === "healthy";
  const color = allOk ? "var(--color-success)" : "var(--color-danger)";
  const label = allOk ? "All systems healthy" : "Health issues detected";

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: "8px",
        padding: "0 12px",
        height: "36px",
        color: "var(--color-text-muted)",
        fontSize: "12px",
      }}
      title={collapsed ? label : undefined}
    >
      <Activity size={16} strokeWidth={1.5} style={{ color, flexShrink: 0 }} />
      {!collapsed && <span style={{ color }}>{label}</span>}
    </div>
  );
}

function Sidebar({
  collapsed,
  onCollapse,
  onClose,
  isMobile,
}: {
  collapsed: boolean;
  onCollapse: () => void;
  onClose: () => void;
  isMobile: boolean;
}) {
  const width = isMobile ? 240 : collapsed ? 60 : 240;

  return (
    <nav
      style={{
        width,
        minWidth: width,
        maxWidth: width,
        background: "var(--color-bg-surface)",
        borderRight: "1px solid var(--color-border-subtle)",
        display: "flex",
        flexDirection: "column",
        transition: "width 200ms ease, min-width 200ms ease, max-width 200ms ease",
        overflow: "hidden",
        position: "fixed",
        top: 0,
        left: 0,
        height: "100vh",
        zIndex: 50,
      }}
    >
      {/* Logo */}
      <div
        style={{
          height: "60px",
          display: "flex",
          alignItems: "center",
          padding: "0 14px",
          borderBottom: "1px solid var(--color-border-subtle)",
          flexShrink: 0,
        }}
      >
        <Link
          to="/"
          style={{ display: "flex", alignItems: "center", textDecoration: "none" }}
        >
          <div
            style={{
              width: 32,
              height: 32,
              borderRadius: "8px",
              background: "var(--color-accent)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              flexShrink: 0,
            }}
          >
            <Film size={18} color="white" strokeWidth={2} />
          </div>
          {(!collapsed || isMobile) && (
            <span
              style={{
                marginLeft: "10px",
                fontSize: "16px",
                fontWeight: 700,
                color: "var(--color-text-primary)",
                letterSpacing: "-0.01em",
                whiteSpace: "nowrap",
                flex: 1,
              }}
            >
              Luminarr
            </span>
          )}
        </Link>
        {isMobile && (
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--color-text-muted)",
              display: "flex",
              alignItems: "center",
              padding: 4,
              marginLeft: "auto",
            }}
          >
            <X size={18} />
          </button>
        )}
      </div>

      {/* Nav items */}
      <div
        style={{
          flex: 1,
          overflowY: "auto",
          overflowX: "hidden",
          padding: "12px 8px",
          display: "flex",
          flexDirection: "column",
          gap: "2px",
        }}
      >
        {mainNav.map((item) => (
          <SidebarNavItem
            key={item.to}
            item={item}
            collapsed={!isMobile && collapsed}
            onClick={isMobile ? onClose : undefined}
          />
        ))}

        <div
          style={{
            margin: "12px 4px 4px",
            fontSize: "11px",
            fontWeight: 500,
            color: "var(--color-text-muted)",
            letterSpacing: "0.08em",
            textTransform: "uppercase",
            whiteSpace: "nowrap",
            overflow: "hidden",
            height: (!isMobile && collapsed) ? "1px" : "auto",
            opacity: (!isMobile && collapsed) ? 0 : 1,
            transition: "opacity 150ms ease",
          }}
        >
          Settings
        </div>

        {settingsNav.map((item) => (
          <SidebarNavItem
            key={item.to}
            item={item}
            collapsed={!isMobile && collapsed}
            onClick={isMobile ? onClose : undefined}
          />
        ))}
      </div>

      {/* Bottom area */}
      <div
        style={{
          borderTop: "1px solid var(--color-border-subtle)",
          padding: "8px",
          display: "flex",
          flexDirection: "column",
          gap: "4px",
        }}
      >
        <HealthDot collapsed={!isMobile && collapsed} />
        {!isMobile && (
          <button
            onClick={onCollapse}
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: collapsed ? "center" : "flex-end",
              width: "100%",
              padding: "0 12px",
              height: "36px",
              background: "none",
              border: "none",
              cursor: "pointer",
              color: "var(--color-text-muted)",
              borderRadius: "6px",
              transition: "background 150ms ease, color 150ms ease",
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = "var(--color-bg-elevated)";
              (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-secondary)";
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = "none";
              (e.currentTarget as HTMLButtonElement).style.color = "var(--color-text-muted)";
            }}
            title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            {collapsed ? (
              <ChevronRight size={16} strokeWidth={1.5} />
            ) : (
              <ChevronLeft size={16} strokeWidth={1.5} />
            )}
          </button>
        )}
      </div>
    </nav>
  );
}

export function Shell() {
  useWebSocket();
  const commandPalette = useCommandPalette();

  // Apply persisted theme once on mount.
  useEffect(() => { applyTheme(); }, []);

  const [collapsed, setCollapsed] = useState(() => {
    return localStorage.getItem("sidebar-collapsed") === "true";
  });
  const [mobileOpen, setMobileOpen] = useState(false);
  const isMobile = useIsMobile();

  // Close mobile sidebar when switching to desktop
  useEffect(() => {
    if (!isMobile) setMobileOpen(false);
  }, [isMobile]);

  useEffect(() => {
    localStorage.setItem("sidebar-collapsed", String(collapsed));
  }, [collapsed]);

  const location = useLocation();
  useEffect(() => {
    window.scrollTo(0, 0);
    setMobileOpen(false);
  }, [location.pathname]);

  const desktopWidth = collapsed ? 60 : 240;

  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      {/* Mobile overlay backdrop */}
      {isMobile && mobileOpen && (
        <div
          onClick={() => setMobileOpen(false)}
          style={{
            position: "fixed",
            inset: 0,
            background: "rgba(0,0,0,0.5)",
            zIndex: 49,
          }}
        />
      )}

      {/* Sidebar wrapper — slides in/out on mobile */}
      <div
        style={{
          transform: isMobile
            ? mobileOpen ? "translateX(0)" : "translateX(-100%)"
            : "none",
          transition: "transform 200ms ease",
        }}
      >
        <Sidebar
          collapsed={collapsed}
          onCollapse={() => setCollapsed((c) => !c)}
          onClose={() => setMobileOpen(false)}
          isMobile={isMobile}
        />
      </div>

      {/* Main content */}
      <main
        style={{
          flex: 1,
          marginLeft: isMobile ? 0 : desktopWidth,
          transition: "margin-left 200ms ease",
          minWidth: 0,
        }}
      >
        {/* Mobile top bar */}
        {isMobile && (
          <div
            style={{
              position: "sticky",
              top: 0,
              zIndex: 40,
              height: 52,
              background: "var(--color-bg-surface)",
              borderBottom: "1px solid var(--color-border-subtle)",
              display: "flex",
              alignItems: "center",
              padding: "0 16px",
              gap: 12,
            }}
          >
            <button
              onClick={() => setMobileOpen(true)}
              style={{
                background: "none",
                border: "none",
                cursor: "pointer",
                color: "var(--color-text-secondary)",
                display: "flex",
                alignItems: "center",
                padding: 4,
                borderRadius: 6,
              }}
            >
              <Menu size={20} />
            </button>
            <Link
              to="/"
              style={{ display: "flex", alignItems: "center", gap: 8, textDecoration: "none" }}
            >
              <div
                style={{
                  width: 24,
                  height: 24,
                  borderRadius: "6px",
                  background: "var(--color-accent)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                }}
              >
                <Film size={14} color="white" strokeWidth={2} />
              </div>
              <span
                style={{
                  fontSize: "15px",
                  fontWeight: 700,
                  color: "var(--color-text-primary)",
                  letterSpacing: "-0.01em",
                }}
              >
                Luminarr
              </span>
            </Link>
          </div>
        )}

        <Outlet />
      </main>
      {commandPalette.isOpen && <CommandPalette onClose={commandPalette.close} />}
    </div>
  );
}
