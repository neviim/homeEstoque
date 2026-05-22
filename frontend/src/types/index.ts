export interface User {
  id: number;
  name: string;
  email: string;
  role?: string;
  status?: string;
  permissions?: string[];
  created_at?: string;
}

export interface Permission {
  key: string;
  label: string;
  description: string;
  category: string;
}

export interface Role {
  id: number;
  name: string;
  label: string;
  description: string;
  is_system: boolean;
  user_count: number;
  permissions: string[];
  created_at?: string;
}

export interface Category {
  id: number;
  name: string;
  parent_id?: number | null;
  icon?: string;
  color?: string;
  item_count?: number;
  created_at?: string;
}

export interface Location {
  id: number;
  name: string;
  type: string;
  parent_id?: number | null;
  description?: string;
  full_path?: string;
  item_count?: number;
  created_at?: string;
}

export interface ItemPhoto {
  id: number;
  item_id: number;
  filename: string;
  original_name?: string;
  size?: number;
  url: string;
  created_at?: string;
}

export interface Item {
  id: number;
  code: string;
  name: string;
  description?: string;
  brand?: string;
  model?: string;
  serial_number?: string;
  quantity: number;
  unit: string;
  purchase_date?: string | null;
  purchase_price?: number | null;
  condition: string;
  notes?: string;
  category_id?: number | null;
  location_id?: number | null;
  category_name?: string;
  location_path?: string;
  photos?: ItemPhoto[];
  created_at?: string;
  updated_at?: string;
}

export interface MovementsPage {
  movements: Movement[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface ItemsPage {
  items: Item[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface Movement {
  id: number;
  item_id: number;
  from_location_id?: number | null;
  to_location_id?: number | null;
  quantity: number;
  reason?: string;
  user_id?: number | null;
  from_location_name?: string;
  to_location_name?: string;
  user_name?: string;
  item_name?: string;
  created_at?: string;
}

export interface DashboardStats {
  total_items: number;
  total_quantity: number;
  total_categories: number;
  total_locations: number;
  total_value: number;
  recent_items: Item[];
  top_categories: Category[];
}
