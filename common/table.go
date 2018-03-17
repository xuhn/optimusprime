package common

import (
	"optimusprime/log"
	"fmt"
	"strings"
)

type Table struct {
	Fields     []string
	Footer     map[string]string
	Rows       []map[string]string
	HideHead   bool // when true doesn't print header
	Markdown   bool
	fieldSizes map[string]int
}

// New - Creates a new table.
func NewTable(fields []string) *Table {
	return &Table{
		Fields:     fields,
		Rows:       make([]map[string]string, 0),
		fieldSizes: make(map[string]int),
	}
}

// PrintTable - Prints table.
func PrintTable(fields []string, rows []map[string]interface{}) {
	table := NewTable(fields)
	for _, r := range rows {
		table.AddRow(r)
	}
	table.Print()
}

// PrintHorizontal - Prints horizontal table from a map.
func PrintHorizontal(m map[string]interface{}) {
	table := NewTable([]string{"Key", "Value"})
	rows := mapToRows(m)
	for _, row := range rows {
		table.AddRow(row)
	}
	table.HideHead = true
	table.Print()
}

// PrintRow - Prints table with only one row.
func PrintRow(fields []string, row map[string]interface{}) {
	table := NewTable(fields)
	table.AddRow(row)
	table.Print()
}

// AddRow - Adds row to the table.
func (t *Table) AddRow(row map[string]interface{}) {
	newRow := make(map[string]string)
	for _, k := range t.Fields {
		v := row[k]
		// If is not nil format
		// else value is empty string
		var val string
		if v == nil {
			val = ""
		} else {
			val = fmt.Sprintf("%v", v)
		}

		newRow[k] = val
	}

	t.calculateSizes(newRow)

	if len(newRow) > 0 {
		t.Rows = append(t.Rows, newRow)
	}
}

// AddFooter - Adds footer to the table.
func (t *Table) AddFooter(footer map[string]string) {
	t.Footer = footer
}

// Print - Prints table.
func (t *Table) Print() {
	if len(t.Rows) == 0 && t.Footer == nil {
		return
	}

	t.calculateSizes(t.Footer)
	var s string = "\n"
	if !t.Markdown {
		s += fmt.Sprintf("%s\n", t.printDash())
	}

	if !t.HideHead {
		s += fmt.Sprintf("%s\n", t.getHead())
		s += fmt.Sprintf("%s\n", t.printTableDash())
	}

	for _, r := range t.Rows {
		s += fmt.Sprintf("%s\n", t.rowString(r))
		if !t.Markdown {
			s += fmt.Sprintf("%s\n", t.printDash())
		}
	}

	if t.Footer != nil {
		s += fmt.Sprintf("%s\n", t.printTableDash())
		s += fmt.Sprintf("%s\n", t.rowString(t.Footer))
		if !t.Markdown {
			s += fmt.Sprintf("%s\n", t.printTableDash())
		}
	}
	log.INFOF("%s", s)
}

// getHead - Returns table header containing fields names.
func (t *Table) getHead() string {
	s := "|"
	for _, name := range t.Fields {
		s += t.fieldString(name, strings.Title(name)) + "|"
	}
	return s
}

// rowString - Creates a string row.
func (t *Table) rowString(row map[string]string) string {
	s := "|"
	for _, name := range t.Fields {
		value := row[name]
		s += t.fieldString(name, value) + "|"
	}
	return s
}

// fieldString - Creates field value string.
func (t *Table) fieldString(name, value string) string {
	value = fmt.Sprintf(" %s ", value)
	spacesLeft := t.fieldSizes[name] - stringWidth(value)
	if spacesLeft > 0 {
		for i := 0; i < spacesLeft; i++ {
			value += " "
		}
	}
	return value
}

// printTableDash - Prints table dash. Markdown or not depending on settings.
func (t *Table) printTableDash() (dash string) {
	if t.Markdown {
		dash = t.printMarkdownDash()
	} else {
		dash = t.printDash()
	}
	return
}

// printDash - Prints dash (on top and header).
func (t *Table) printDash() (dash string) {
	dash = "|"
	for i := 0; i < t.lineLength()-2; i++ {
		dash += "-"
	}
	dash += "|"
	dash += "\n"
	return
}

// printMarkdownDash - Prints dash in middle of table.
func (t *Table) printMarkdownDash() (dash string) {
	r := make(map[string]string)
	for _, name := range t.Fields {
		r[name] = strings.Repeat("-", t.fieldSizes[name]-2)
	}
	dash = t.rowString(r)
	dash += "\n"
	return
}

// lineLength - Counts size of table line length (with spaces etc.).
func (t *Table) lineLength() (sum int) {
	for _, l := range t.fieldSizes {
		sum += l + 1
	}
	return sum + 1
}

func (t *Table) calculateSizes(row map[string]string) {
	for _, k := range t.Fields {
		v, ok := row[k]
		if !ok {
			continue
		}

		vlen := stringWidth(v)
		// align to field name length
		if klen := stringWidth(k); vlen < klen {
			vlen = klen
		}
		vlen += 2 // + 2 spaces
		if t.fieldSizes[k] < vlen {
			t.fieldSizes[k] = vlen
		}
	}
}

func mapToRows(m map[string]interface{}) (rows []map[string]interface{}) {
	rows = []map[string]interface{}{}
	for key, value := range m {
		row := map[string]interface{}{}
		row["Key"] = strings.Title(key)
		row["Value"] = value
		rows = append(rows, row)
	}
	return
}
