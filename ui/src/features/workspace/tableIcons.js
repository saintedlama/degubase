export const TABLE_ICONS = [
  { name: 'database',    icon: 'ri-database-2-line',         label: 'Database' },
  { name: 'table',       icon: 'ri-table-line',              label: 'Table' },
  { name: 'folder',      icon: 'ri-folder-line',             label: 'Folder' },
  { name: 'list',        icon: 'ri-file-list-3-line',        label: 'List' },
  { name: 'contacts',    icon: 'ri-contacts-book-line',      label: 'Contacts' },
  { name: 'user',        icon: 'ri-user-line',               label: 'User' },
  { name: 'team',        icon: 'ri-team-line',               label: 'Team' },
  { name: 'building',    icon: 'ri-building-line',           label: 'Building' },
  { name: 'calendar',    icon: 'ri-calendar-line',           label: 'Calendar' },
  { name: 'task',        icon: 'ri-task-line',               label: 'Task' },
  { name: 'checkbox',    icon: 'ri-checkbox-circle-line',    label: 'Checkbox' },
  { name: 'bookmark',    icon: 'ri-bookmark-line',           label: 'Bookmark' },
  { name: 'star',        icon: 'ri-star-line',               label: 'Star' },
  { name: 'heart',       icon: 'ri-heart-line',              label: 'Heart' },
  { name: 'mail',        icon: 'ri-mail-line',               label: 'Mail' },
  { name: 'chat',        icon: 'ri-chat-1-line',             label: 'Chat' },
  { name: 'chart-bar',   icon: 'ri-bar-chart-line',          label: 'Bar chart' },
  { name: 'chart-pie',   icon: 'ri-pie-chart-line',          label: 'Pie chart' },
  { name: 'money',       icon: 'ri-money-dollar-circle-line', label: 'Money' },
  { name: 'cart',        icon: 'ri-shopping-cart-line',      label: 'Cart' },
  { name: 'code',        icon: 'ri-code-s-slash-line',       label: 'Code' },
  { name: 'settings',    icon: 'ri-settings-3-line',         label: 'Settings' },
  { name: 'globe',       icon: 'ri-global-line',             label: 'Globe' },
  { name: 'location',    icon: 'ri-map-pin-line',            label: 'Location' },
  { name: 'truck',       icon: 'ri-truck-line',              label: 'Truck' },
  { name: 'flask',       icon: 'ri-flask-line',              label: 'Flask' },
  { name: 'graduation',  icon: 'ri-graduation-cap-line',     label: 'Graduation' },
  { name: 'rocket',      icon: 'ri-rocket-line',             label: 'Rocket' },
  { name: 'lightning',   icon: 'ri-flashlight-line',         label: 'Lightning' },
  { name: 'shield',      icon: 'ri-shield-line',             label: 'Shield' },
  { name: 'document',    icon: 'ri-file-text-line',          label: 'Document' },
  { name: 'tag',         icon: 'ri-price-tag-3-line',        label: 'Tag' },
  { name: 'phone',       icon: 'ri-phone-line',              label: 'Phone' },
  { name: 'home',        icon: 'ri-home-line',               label: 'Home' },
]

const _nameToIcon = Object.fromEntries(TABLE_ICONS.map(i => [i.name, i.icon]))

/**
 * Resolve a stored icon value to a Remix Icon CSS class.
 * Accepts semantic names ("database") or legacy CSS class names ("ri-database-2-line").
 */
export function resolveTableIcon(nameOrClass) {
  if (!nameOrClass) return ''
  if (_nameToIcon[nameOrClass]) return _nameToIcon[nameOrClass]
  if (nameOrClass.startsWith('ri-')) return nameOrClass  // backward compat
  return ''
}
