package provider

import (
	"fmt"
	"sort"
	"strings"
)

// Resources and DataSources list what the provider offers, for the documentation generator.
func Resources() []*Resource { return allResources() }

func DataSources() []*DataSource { return allDataSources() }

var kindNames = map[Kind]string{String: "String", Int: "Number", Bool: "Boolean", StringList: "List of String", StringMap: "Map of String", List: "Attributes List", Object: "Attributes"}

func attrLine(a Attr, depth int) string {
	desc := a.Desc
	if a.Enum != "" && !strings.Contains(desc, "values") {
		if desc != "" && !strings.HasSuffix(desc, ".") {
			desc += "."
		}
	}
	notes := []string{}
	if a.ForceNew {
		notes = append(notes, "changing it replaces the resource")
	}
	if a.Sensitive {
		notes = append(notes, "sensitive")
	}
	if a.Local {
		notes = append(notes, "not read back from the API")
	}
	text := strings.TrimSpace(desc)
	if len(notes) > 0 {
		if text != "" && !strings.HasSuffix(text, ".") {
			text += "."
		}
		text += " (" + strings.Join(notes, "; ") + ")"
	}
	return fmt.Sprintf("%s- `%s` (%s) %s", strings.Repeat("  ", depth), a.Name, kindNames[a.Kind], strings.TrimSpace(text))
}

func writeAttrs(b *strings.Builder, attrs []Attr, depth int) {
	for _, a := range attrs {
		b.WriteString(attrLine(a, depth))
		b.WriteString("\n")
		if len(a.Nested) > 0 {
			writeAttrs(b, a.Nested, depth+1)
		}
	}
}

func section(b *strings.Builder, title string, attrs []Attr, pick func(Attr) bool) {
	var list []Attr
	for _, a := range attrs {
		if pick(a) {
			list = append(list, a)
		}
	}
	if len(list) == 0 {
		return
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	b.WriteString("### " + title + "\n\n")
	writeAttrs(b, list, 0)
	b.WriteString("\n")
}

// ResourceDoc is the page of a resource for the registry.
func ResourceDoc(r *Resource, example string) string {
	var b strings.Builder
	name := "adatage_" + r.Name
	fmt.Fprintf(&b, "---\npage_title: \"%s Resource - terraform-provider-adatage\"\nsubcategory: \"\"\ndescription: |-\n  %s\n---\n\n# %s (Resource)\n\n%s\n\n", name, firstLine(r.Desc), name, r.Desc)
	if example != "" {
		b.WriteString("## Example Usage\n\n```terraform\n" + strings.TrimSpace(example) + "\n```\n\n")
	}
	b.WriteString("## Schema\n\n")
	section(&b, "Required", r.Attrs, func(a Attr) bool { return a.Required })
	section(&b, "Optional", r.Attrs, func(a Attr) bool { return !a.Required && a.Optional })
	section(&b, "Read-Only", r.Attrs, func(a Attr) bool { return !a.Required && !a.Optional })
	b.WriteString("## Import\n\n")
	switch {
	case r.ImportParse != nil:
		fmt.Fprintf(&b, "```shell\nterraform import %s.example zones/ZONE/disks/DISK\n```\n", name)
	default:
		fmt.Fprintf(&b, "```shell\nterraform import %s.example %s\n```\n", name, strings.ToUpper(strings.Join(r.importFields(), "/")))
	}
	return b.String()
}

// DataSourceDoc is the page of a data source.
func DataSourceDoc(d *DataSource, example string) string {
	var b strings.Builder
	name := "adatage_" + d.Name
	fmt.Fprintf(&b, "---\npage_title: \"%s Data Source - terraform-provider-adatage\"\nsubcategory: \"\"\ndescription: |-\n  %s\n---\n\n# %s (Data Source)\n\n%s\n\n", name, firstLine(d.Desc), name, d.Desc)
	if example != "" {
		b.WriteString("## Example Usage\n\n```terraform\n" + strings.TrimSpace(example) + "\n```\n\n")
	}
	b.WriteString("## Schema\n\n")
	section(&b, "Required", d.Attrs, func(a Attr) bool { return a.Required })
	section(&b, "Optional", d.Attrs, func(a Attr) bool { return !a.Required && a.Optional })
	section(&b, "Read-Only", d.Attrs, func(a Attr) bool { return !a.Required && !a.Optional })
	return b.String()
}

func firstLine(s string) string {
	if i := strings.Index(s, ". "); i > 0 {
		return s[:i+1]
	}
	return s
}
