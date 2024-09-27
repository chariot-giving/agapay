interface SidebarProps {
  isCollapsed: boolean;
}

export default function Sidebar({ isCollapsed }: SidebarProps) {
  const menuItems = [
    { icon: '📊', label: 'Overview', active: true },
    { icon: '📅', label: 'Activity' },
    { icon: '🏛️', label: 'Nonprofits' },
    { icon: '💰', label: 'Billing' },
    { icon: '⚙️', label: 'Settings' },
  ];

  return (
    <aside className={`bg-gray-100 h-screen transition-all duration-300 ${isCollapsed ? 'w-16' : 'w-64'}`}>
      <nav className="mt-5">
        {menuItems.map((item, index) => (
          <a
            key={index}
            href={`/${item.label.toLowerCase()}`}
            className={`flex items-center px-6 py-2 mt-4 duration-200 border-l-4 ${
              item.active
                ? 'border-blue-500 bg-blue-100 text-blue-500'
                : 'border-transparent hover:bg-gray-200 hover:border-gray-300'
            }`}
          >
            <span className="text-lg mr-4">{item.icon}</span>
            {!isCollapsed && <span className="text-sm font-medium text-black">{item.label}</span>}
          </a>
        ))}
      </nav>
    </aside>
  );
}
