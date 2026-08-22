package adapter

type Meta struct {
	Engine    string `json:"engine"`
	Version   string `json:"version"`
	Status    string `json:"status"`
	Schemas   int    `json:"schemas"`
	Tables    int    `json:"tables"`
	Views     int    `json:"views"`
	Functions int    `json:"functions"`
}

type Object struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ObjectSummary struct {
	Tables     int `json:"tables"`
	Views      int `json:"views"`
	MatViews   int `json:"mat_views"`
	Functions  int `json:"functions"`
	Procedures int `json:"procedures"`
	Triggers   int `json:"triggers"`
	Sequences  int `json:"sequences"`
	Types      int `json:"types"`
	Extensions int `json:"extensions"`
}

type Column struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Nullable  bool    `json:"nullable"`
	Default   *string `json:"default"`
	Primary   bool    `json:"primary_key"`
	Unique    bool    `json:"unique"`
	Identity  string  `json:"identity"`
	Generated string  `json:"generated"`
	Collation *string `json:"collation"`
	Comment   string  `json:"comment"`
}

type Index struct {
	Name      string   `json:"name"`
	Method    string   `json:"method"`
	Unique    bool     `json:"unique"`
	Columns   []string `json:"columns"`
	Predicate string   `json:"predicate"`
	Primary   bool     `json:"primary"`
}

type Constraint struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Column     string `json:"column"`
	RefTable   string `json:"ref_table,omitempty"`
	RefColumn  string `json:"ref_column,omitempty"`
	Definition string `json:"definition"`
}

type ForeignKey struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	RefTable   string   `json:"ref_table"`
	RefColumns []string `json:"ref_columns"`
	OnDelete   string   `json:"on_delete"`
	OnUpdate   string   `json:"on_update"`
}

type Trigger struct {
	Name     string `json:"name"`
	Event    string `json:"event"`
	Timing   string `json:"timing"`
	Function string `json:"function"`
	Enabled  string `json:"enabled"`
}

type TableDetail struct {
	Schema      string       `json:"schema"`
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	Owner       string       `json:"owner"`
	Columns     []Column     `json:"columns"`
	Indexes     []Index      `json:"indexes"`
	Constraints []Constraint `json:"constraints"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
	Triggers    []Trigger    `json:"triggers"`
}

type Filter struct {
	Column string `json:"column"`
	Op     string `json:"op"`
	Value  string `json:"value"`
}

type QueryOptions struct {
	Limit   int
	Offset  int
	Sort    string
	Order   string
	Filters []Filter
	Timeout int
	MaxRows int
	Schema  []Column
}

type QueryResult struct {
	Columns    []string        `json:"columns"`
	Rows       [][]interface{} `json:"rows"`
	RowCount   int             `json:"row_count"`
	DurationMs int64           `json:"duration_ms"`
	ReadOnly   bool            `json:"read_only"`
	Truncated  bool            `json:"truncated"`
	Message    string          `json:"message,omitempty"`
	Error      *QueryError     `json:"error,omitempty"`
}

type QueryError struct {
	Message    string `json:"message"`
	Position   int    `json:"position,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *QueryError) Error() string { return e.Message }

type ExecResult struct {
	Message    string `json:"message"`
	CommandTag string `json:"command_tag"`
	DurationMs int64  `json:"duration_ms"`
}
