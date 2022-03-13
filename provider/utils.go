package provider

import (
  "strings"
  "encoding/base64"
  "math/rand"
)

func sorted_list_3way(a []string, b []string) (left, inner, right []string) {
  na := len(a)
  nb := len(b)
  i := 0
  j := 0

  for i < na && j < nb {
    va := a[i]
    vb := b[j]
    cmp := strings.Compare(va, vb)
    if cmp < 0 {
      left = append(left, va)
      i += 1
    } else if cmp > 0 {
      right = append(right, vb)
      j += 1
    } else {
      inner = append(inner, va)
      i += 1
      j += 1
    }
  }
  for i < na {
    left = append(left, a[i])
    i += 1
  }
  for j < nb {
    right = append(right, b[j])
    j += 1
  }

  return
}

func generate_id() string {
  var bytes [12]byte
  rand.Read(bytes[:])
  return base64.StdEncoding.EncodeToString(bytes[:])
}
