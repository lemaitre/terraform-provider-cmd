package cmd

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"

	//"github.com/hashicorp/terraform-plugin-framework/diag"
	//"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	//"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	//"github.com/hashicorp/terraform-plugin-log/tflog"
)

func formatList(t string, val tftypes.Value, out io.Writer, prefix string) {
	fmt.Fprintf(out, "%s(", t)
	if val.IsNull() {
		fmt.Fprintf(out, "null")
	} else if !val.IsKnown() {
		fmt.Fprintf(out, "unknown")
	} else {
		var l []tftypes.Value
		val.As(&l)
		for _, v := range l {
			fmt.Fprintf(out, "\n%s  ", prefix)
			formatVal_(v, out, prefix+"  ")
		}
		fmt.Fprintf(out, "\n%s", prefix)
	}
	fmt.Fprintf(out, ")")
}

func formatMap(t string, val tftypes.Value, out io.Writer, prefix string) {
	fmt.Fprintf(out, "%s(", t)
	if val.IsNull() {
		fmt.Fprintf(out, "null")
	} else if !val.IsKnown() {
		fmt.Fprintf(out, "unknown")
	} else {
		var m map[string]tftypes.Value
		keys := []string{}
		val.As(&m)
		for k, _ := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v, _ := m[k]
			fmt.Fprintf(out, "\n%s  %s = ", prefix, k)
			formatVal_(v, out, prefix+"  ")
		}
		fmt.Fprintf(out, "\n%s", prefix)
	}
	fmt.Fprintf(out, ")")
}

func formatVal_(val tftypes.Value, out io.Writer, prefix string) {
	t := val.Type()

	if t.Is(tftypes.List{}) {
		formatList("List", val, out, prefix)
	} else if t.Is(tftypes.Set{}) {
		formatList("Set", val, out, prefix)
	} else if t.Is(tftypes.Tuple{}) {
		formatList("Tuple", val, out, prefix)
	} else if t.Is(tftypes.Map{}) {
		formatMap("Map", val, out, prefix)
	} else if t.Is(tftypes.Object{}) {
		formatMap("Object", val, out, prefix)
	} else if t.Equal(tftypes.String) {
		if val.IsNull() {
			fmt.Fprintf(out, "String(null)")
		} else if !val.IsKnown() {
			fmt.Fprintf(out, "String(unknown)")
		} else {
			var s string
			val.As(&s)
			fmt.Fprintf(out, "%s", strconv.Quote(s))
		}
	} else if t.Equal(tftypes.Number) {
		if val.IsNull() {
			fmt.Fprintf(out, "Number(null)")
		} else if !val.IsKnown() {
			fmt.Fprintf(out, "Number(unknown)")
		} else {
			var f float64
			val.As(&f)
			fmt.Fprintf(out, "%g", f)
		}
	} else if t.Equal(tftypes.Bool) {
		if val.IsNull() {
			fmt.Fprintf(out, "Bool(null)")
		} else if !val.IsKnown() {
			fmt.Fprintf(out, "Bool(unknown)")
		} else {
			var b bool
			val.As(&b)
			if b {
				fmt.Fprintf(out, "true")
			} else {
				fmt.Fprintf(out, "false")
			}
		}
	} else {
		if val.IsNull() {
			fmt.Fprintf(out, "Unknown(null)")
		} else if !val.IsKnown() {
			fmt.Fprintf(out, "Unknown(unknown)")
		} else {
			fmt.Fprintf(out, "Unknown(%s)", val)
		}
	}
}

func formatVal(val tftypes.Value) string {
	var out bytes.Buffer
	formatVal_(val, &out, "")
	return out.String()
}
