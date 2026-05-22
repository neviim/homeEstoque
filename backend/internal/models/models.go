package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	ParentID  *int64    `json:"parent_id,omitempty"`
	Icon      string    `json:"icon,omitempty"`
	Color     string    `json:"color,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ItemCount int       `json:"item_count,omitempty"`
}

type Location struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	ParentID    *int64    `json:"parent_id,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	FullPath    string    `json:"full_path,omitempty"`
	ItemCount   int       `json:"item_count,omitempty"`
}

type Item struct {
	ID            int64     `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	Brand         string    `json:"brand,omitempty"`
	Model         string    `json:"model,omitempty"`
	SerialNumber  string    `json:"serial_number,omitempty"`
	Quantity      int       `json:"quantity"`
	Unit          string    `json:"unit"`
	PurchaseDate  *string   `json:"purchase_date,omitempty"`
	PurchasePrice *float64  `json:"purchase_price,omitempty"`
	Condition     string    `json:"condition"`
	Notes         string    `json:"notes,omitempty"`
	CategoryID    *int64    `json:"category_id,omitempty"`
	LocationID    *int64    `json:"location_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	CategoryName string       `json:"category_name,omitempty"`
	LocationPath string       `json:"location_path,omitempty"`
	Photos       []ItemPhoto  `json:"photos,omitempty"`
}

type ItemPhoto struct {
	ID           int64     `json:"id"`
	ItemID       int64     `json:"item_id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"original_name,omitempty"`
	Size         int64     `json:"size,omitempty"`
	URL          string    `json:"url"`
	CreatedAt    time.Time `json:"created_at"`
}

type Movement struct {
	ID             int64     `json:"id"`
	ItemID         int64     `json:"item_id"`
	FromLocationID *int64    `json:"from_location_id,omitempty"`
	ToLocationID   *int64    `json:"to_location_id,omitempty"`
	Quantity       int       `json:"quantity"`
	Reason         string    `json:"reason,omitempty"`
	UserID         *int64    `json:"user_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`

	FromLocationName string `json:"from_location_name,omitempty"`
	ToLocationName   string `json:"to_location_name,omitempty"`
	UserName         string `json:"user_name,omitempty"`
	ItemName         string `json:"item_name,omitempty"`
}

type DashboardStats struct {
	TotalItems      int     `json:"total_items"`
	TotalQuantity   int     `json:"total_quantity"`
	TotalCategories int     `json:"total_categories"`
	TotalLocations  int     `json:"total_locations"`
	TotalValue      float64 `json:"total_value"`
	RecentItems     []Item  `json:"recent_items"`
	TopCategories   []Category `json:"top_categories"`
}
