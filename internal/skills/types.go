package skills

type Skill struct {
	ID          string
	Category    string
	Slug        string
	Path        string
	Name        string
	Description string
	Deprecated  bool
	ReplacedBy  string
}
