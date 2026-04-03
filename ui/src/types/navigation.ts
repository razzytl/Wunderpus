export type ViewType = 'chat' | 'terminal' | 'database' | 'activity';

export interface NavItem {
  id: ViewType;
  label: string;
  icon: React.ReactNode;
  description: string;
}
