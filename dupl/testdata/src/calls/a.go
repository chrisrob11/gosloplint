package fixture

type flags struct{}

func (flags) StringVar(*string, string, string, string) {}
func (flags) Handle(string, []string, func())           {}
func (flags) Sub() flags                                { return flags{} }

type options struct{ name, dir, mode, user, host, port string }

func register(fs flags, o *options) {
	fs.StringVar(&o.name, "name", "", "the name")
	fs.StringVar(&o.dir, "dir", ".", "the directory")
	fs.StringVar(&o.mode, "mode", "fast", "the mode")
	fs.StringVar(&o.user, "user", "", "the user")
	fs.StringVar(&o.host, "host", "localhost", "the host")
	fs.Sub().Handle(o.port, []string{"a"}, func() {})
}

func set(int) {}

type counts struct {
	a, b, c, d, e, f, g int
	run                 []func(int)
}

func bump(c counts) {
	set(c.a*2 + 1)
	set(c.b*2 + 1)
	set(c.c*2 + 1)
	set(c.d*2 + 1)
	set(c.e*2 + 1)
	set(c.f*2 + 1)
	set(c.g*2 + 1)
	c.run[0](c.a)
}
