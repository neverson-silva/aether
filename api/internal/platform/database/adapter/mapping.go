package adapter

import (
	"strconv"
	"strings"
)

func toInt(v interface{}) int {
	switch t := v.(type) {
	case int64:
		return int(t)
	case int:
		return t
	case float64:
		return int(t)
	case bool:
		if t {
			return 1
		}
		return 0
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func toBool(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case int64:
		return t != 0
	case int:
		return t != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "t", "true", "yes", "1", "on":
			return true
		}
	}
	return false
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func splitList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func mapColumns(rows [][]interface{}) []Column {
	out := make([]Column, 0, len(rows))
	for _, r := range rows {
		c := Column{Name: cellString(r[0]), Type: cellString(r[1]), Nullable: toBool(r[2])}
		c.Default = ptr(cellString(r[3]))
		c.Identity = cellString(r[4])
		c.Generated = cellString(r[5])
		c.Comment = cellString(r[6])
		c.Primary = toBool(r[7])
		c.Unique = toBool(r[8])
		out = append(out, c)
	}
	return out
}

func mapIndexes(rows [][]interface{}) []Index {
	out := make([]Index, 0, len(rows))
	for _, r := range rows {
		out = append(out, Index{
			Name:      cellString(r[0]),
			Method:    cellString(r[1]),
			Unique:    toBool(r[2]),
			Primary:   toBool(r[3]),
			Columns:   splitList(cellString(r[4])),
			Predicate: cellString(r[5]),
		})
	}
	return out
}

func mapConstraints(rows [][]interface{}) []Constraint {
	out := make([]Constraint, 0, len(rows))
	for _, r := range rows {
		out = append(out, Constraint{
			Name:       cellString(r[0]),
			Type:       constraintLabel(cellString(r[1])),
			Column:     cellString(r[2]),
			Definition: cellString(r[3]),
		})
	}
	return out
}

func constraintLabel(typ string) string {
	switch typ {
	case "p":
		return "PRIMARY KEY"
	case "u":
		return "UNIQUE"
	case "c":
		return "CHECK"
	case "n":
		return "NOT NULL"
	default:
		return typ
	}
}

func mapForeignKeys(rows [][]interface{}) []ForeignKey {
	out := make([]ForeignKey, 0, len(rows))
	for _, r := range rows {
		out = append(out, ForeignKey{
			Name:       cellString(r[0]),
			Columns:    splitList(cellString(r[1])),
			RefTable:   cellString(r[2]),
			RefColumns: splitList(cellString(r[3])),
			OnDelete:   fkAction(cellString(r[4])),
			OnUpdate:   fkAction(cellString(r[5])),
		})
	}
	return out
}

func fkAction(c string) string {
	switch c {
	case "a":
		return "NO ACTION"
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default:
		return c
	}
}

func mapTriggers(rows [][]interface{}) []Trigger {
	out := make([]Trigger, 0, len(rows))
	for _, r := range rows {
		out = append(out, Trigger{
			Name:     cellString(r[0]),
			Timing:   cellString(r[1]),
			Event:    cellString(r[2]),
			Function: cellString(r[3]),
			Enabled:  cellString(r[4]),
		})
	}
	return out
}
