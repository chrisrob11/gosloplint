// Package flags reads its module from the driver.
package flags

import (
	"example.com/app/shape"
	"example.com/cli"
)

// Mixed holds two types of another module's interface.
func Mixed() []cli.Flag { return []cli.Flag{&cli.StringFlag{}, &cli.BoolFlag{}} }

// Shapes holds two types of an interface of this module, which it can change.
func Shapes() []shape.Shape { return []shape.Shape{shape.Square{}, shape.Circle{}} } // want `returns-interface: Shapes`
