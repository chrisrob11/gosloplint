package a

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiteralFields(t *testing.T) {
	c := Config{Name: "db", Port: 5432}
	assert.Equal(t, "db", c.Name) // want `literal-field: c.Name was set to "db" on line 11 and nothing since could change it`
	n := len(c.Tags)
	require.Equal(t, c.Port, 5432) // want `literal-field: c.Port was set to 5432 on line 11`
	assert.True(t, c.Port == 5432) // want `literal-field: c.Port was set to 5432 on line 11`
	if c.Name != "db" {            // want `literal-field: c.Name was set to "db" on line 11`
		t.Fatal(n)
	}

	p := &Config{}
	p.Port = 1
	p.Name = "x"
	assert.Equal(t, 1, p.Port) // want `literal-field: p.Port was set to 1 on line 21`
}

// Something between the set and the assertion may change the field, or the
// assertion compares another value.
func TestLiteralFieldsChanged(t *testing.T) {
	c := Config{Port: 1}
	c.Normalize()
	assert.Equal(t, 1, c.Port)

	d := Config{Port: 1}
	reset(&d)
	assert.Equal(t, 1, d.Port)

	e := Config{Port: 1}
	e.Port++
	assert.Equal(t, 1, e.Port)

	f := Config{Port: 1}
	assert.Equal(t, 2, f.Port)

	g := Config{Port: 1}
	if len(query()) > 0 {
		g.Port = 2
	}
	assert.Equal(t, 1, g.Port)

	h := Config{Port: 1}
	h = Config{Port: 2}
	assert.Equal(t, 1, h.Port)

	i := Config{Port: 1, Name: name()}
	assert.Equal(t, 1, i.Port)

	j := Config{}
	j.Port = 1
	k := &j
	k.Port = 2
	assert.Equal(t, 1, j.Port)

	l := Config{Port: 1}
	ch := make(chan int)
	<-ch
	assert.Equal(t, 1, l.Port)

	m := Config{Port: 1}
	m.Port = 3
	assert.Equal(t, 1, m.Port)
}

func reset(c *Config) { c.Port = 0 }

func name() string { return "n" }
