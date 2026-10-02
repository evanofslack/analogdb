package analogdb

// CatalogPost is a top scoring post shown with a film or camera
type CatalogPost struct {
	Id     int     `json:"id" example:"1"`
	Title  string  `json:"title" example:"Sunset over the lake"`
	Score  int     `json:"score" example:"150"`
	Images []Image `json:"images"`
}
