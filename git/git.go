package git

import (
	"context"
	"io"
	"os"
	"os/exec"
)

type Cmd struct {
	beforeExec []func() error
	cmdBuf     []string
	ctx        context.Context
}

func Git() *Cmd {
	c := new(Cmd)
	c.beforeExec = make([]func() error, 0, 1)
	c.cmdBuf = make([]string, 0, 10)
	c.cmdBuf = append(c.cmdBuf, "git")
	return c
}

func (c *Cmd) Clone(remote string, dest string) *Cmd {
	if dest != "" {
		c.beforeExec = append(c.beforeExec, func() error {
			return os.MkdirAll(dest, 0o777)
		})
	}
	c.cmdBuf = append(c.cmdBuf, "clone")
	c.cmdBuf = append(c.cmdBuf, "--depth")
	c.cmdBuf = append(c.cmdBuf, "1")
	c.cmdBuf = append(c.cmdBuf, "--no-checkout")
	c.cmdBuf = append(c.cmdBuf, remote)
	if dest != "" {
		c.cmdBuf = append(c.cmdBuf, dest)
	}
	return c
}

func (c *Cmd) Checkout(tree string) *Cmd {
	c.cmdBuf = append(c.cmdBuf, "checkout")
	c.cmdBuf = append(c.cmdBuf, tree)
	return c
}

func (c *Cmd) Sp(paths ...string) *Cmd {
	c.cmdBuf = append(c.cmdBuf, "sparse-checkout")
	c.cmdBuf = append(c.cmdBuf, "set")
	c.cmdBuf = append(c.cmdBuf, "--no-cone")
	c.cmdBuf = append(c.cmdBuf, paths...)
	return c
}

func (c *Cmd) Exec(ctx context.Context, stderr io.Writer) ([]byte, string, error) {
	c.ctx = ctx
	for _, fn := range c.beforeExec {
		err := fn()
		if err != nil {
			return nil, "", err
		}
	}
	cmd := exec.CommandContext(ctx, c.cmdBuf[0], c.cmdBuf[1:]...)
	cmd.Stderr = stderr
	res, err := cmd.Output()
	return res, cmd.String(), err
}
