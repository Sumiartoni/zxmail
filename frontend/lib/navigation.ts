export const navigationSections = [
  {
    title: "Workspace",
    items: [
      { href: "/dashboard", label: "Dashboard", note: "customer overview" },
      { href: "/domains", label: "Domains", note: "dns onboarding" },
      { href: "/domains/new", label: "New Domain", note: "wizard" },
      { href: "/credentials", label: "Credentials", note: "show once secrets" },
      { href: "/logs", label: "Logs", note: "event explorer" },
      { href: "/suppressions", label: "Suppressions", note: "recipient safety" },
    ],
  },
  {
    title: "Admin",
    items: [
      { href: "/admin", label: "Overview", note: "operator view" },
      { href: "/admin/customers", label: "Customers", note: "organizations" },
      { href: "/admin/logs", label: "Admin Logs", note: "global filters" },
      { href: "/admin/system", label: "System", note: "health and guardrails" },
    ],
  },
];
