package fixture

func routes(fs flags, o *options) {
	fs.StringVar(&o.port, "port", "80", "the port")
	fs.StringVar(&o.user, "login", "", "the login")
	fs.StringVar(&o.host, "addr", "", "the address")
	fs.StringVar(&o.dir, "root", "/", "the root")
	fs.StringVar(&o.mode, "level", "", "the level")
	fs.Sub().Handle(o.name, []string{"b"}, func() {})
}

func lift(c counts) {
	set(c.g*2 + 1) // want `lines 13-20 duplicate lines 28-35 of a.go \(bump\)`
	set(c.f*2 + 1)
	set(c.e*2 + 1)
	set(c.d*2 + 1)
	set(c.c*2 + 1)
	set(c.b*2 + 1)
	set(c.a*2 + 1)
	c.run[0](c.a)
}

func again(fs flags, p *options) {
	fs.StringVar(&p.dir, "name", "", "the name") // want `lines 24-29 duplicate lines 12-17 of a.go \(register\)`
	fs.StringVar(&p.name, "dir", ".", "the directory")
	fs.StringVar(&p.user, "mode", "fast", "the mode")
	fs.StringVar(&p.mode, "user", "", "the user")
	fs.StringVar(&p.port, "host", "localhost", "the host")
	fs.Sub().Handle(p.host, []string{"a"}, func() {})
}
